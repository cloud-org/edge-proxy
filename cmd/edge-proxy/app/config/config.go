package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/config"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/projectinfo"
)

// EdgeProxyConfiguration represents configuration of edge proxy
type EdgeProxyConfiguration struct {
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

	cfg := &EdgeProxyConfiguration{
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

	//  modify content-type 貌似没啥用
	//cfg.AcceptContentTypes = "application/vnd.kubernetes.protobuf"
	//cfg.ContentType = "application/vnd.kubernetes.protobuf"
	//klog.Infof("cfg content-type %v, %v", cfg.AcceptContentTypes, cfg.ContentType)

	return rest.TransportFor(cfg)
}
