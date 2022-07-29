package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/openyurtio/openyurt/pkg/yurthub/storage"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/config"
	"github.com/imroc/req/v3"
	"github.com/openyurtio/openyurt/pkg/yurthub/storage/factory"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/projectinfo"
)

// EdgeProxyConfiguration represents configuration of edge proxy
type EdgeProxyConfiguration struct {
	StorageMgr          storage.Store
	RT                  http.RoundTripper
	RemoteServers       []*url.URL
	DiskCachePath       string
	BindAddr            string
	EdgeProxyServerAddr string
	EnableSampleHandler bool
}

// Complete converts *options.BenchMarkOptions to *EdgeProxyConfiguration
func Complete(options *options.EdgeProxyOptions) (*EdgeProxyConfiguration, error) {
	// 解析成 []*url.URL 切片
	us, err := parseRemoteServers(options.ServerAddr)
	if err != nil {
		return nil, err
	}

	// 获取 roundTripper 表示执行单个HTTP事务的能力，获得给定请求的响应
	rt, err := prepareRoundTripper(options.UseKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("could not new round tripper, %w", err)
	}

	// create disk storage
	storageManager, err := factory.CreateStorage(options.DiskCachePath)
	if err != nil {
		klog.Errorf("could not create storage manager, %v", err)
		return nil, err
	}

	cfg := &EdgeProxyConfiguration{
		StorageMgr:          storageManager,
		RT:                  rt,
		RemoteServers:       us,
		DiskCachePath:       options.DiskCachePath,
		BindAddr:            net.JoinHostPort("127.0.0.1", "10267"),
		EdgeProxyServerAddr: net.JoinHostPort("127.0.0.1", "10261"),
		EnableSampleHandler: options.EnableSampleHandler,
	}

	return cfg, nil
}

func parseRemoteServers(serverAddr string) ([]*url.URL, error) {
	if serverAddr == "" {
		return make([]*url.URL, 0), fmt.Errorf("--server-addr should be set for edge proxy")
	}
	servers := strings.Split(serverAddr, ",")
	us := make([]*url.URL, 0, len(servers))
	remoteServers := make([]string, 0, len(servers))
	for _, server := range servers {
		u, err := url.Parse(server)
		if err != nil {
			klog.Errorf("failed to parse server address %s, %v", servers, err)
			return us, err
		}
		if u.Scheme == "" {
			u.Scheme = "https"
		} else if u.Scheme != "https" {
			return us, fmt.Errorf("only https scheme is supported for server address(%s)", serverAddr)
		}
		us = append(us, u)
		remoteServers = append(remoteServers, u.String())
	}

	if len(us) < 1 {
		return us, fmt.Errorf("no server address is set, can not connect remote server")
	}
	klog.Infof("%s would connect remote servers: %s", projectinfo.GetProxyName(), strings.Join(remoteServers, ","))

	return us, nil
}

func prepareRoundTripper(usekubeconfig bool) (http.RoundTripper, error) {
	var (
		cfg *rest.Config
		err error
	)
	if usekubeconfig {
		cfg, err = config.GetRestConf()
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	// inject req client
	//tran, err := UseReqClient(cfg)
	//if err != nil {
	//	klog.Errorf("get transport err: %v", err)
	//	return nil, err
	//}
	//cfg.Transport = tran
	//cfg.TLSClientConfig = rest.TLSClientConfig{}

	return rest.TransportFor(cfg)
}

// UseReqClient 注入第三方 req client
func UseReqClient(cfg *rest.Config) (http.RoundTripper, error) {
	// https://req.cool/zh/docs/examples/integrate-with-client-go/
	reqClient := req.C()
	//reqClient.EnableDumpAll()
	reqClient.EnableDebugLog()

	tlsConfig, err := rest.TLSConfigFor(cfg)
	if err != nil {
		klog.Errorf("get tlsConfig err: %v", err)
		return nil, err
	}

	reqTransport := reqClient.GetTransport()
	reqTransport.SetTLSClientConfig(tlsConfig)

	return reqTransport, nil
}
