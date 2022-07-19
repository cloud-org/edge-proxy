package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"
	"code.aliyun.com/openyurt/edge-proxy/pkg/projectinfo"
)

// EdgeProxyConfiguration represents configuration of edge proxy
type EdgeProxyConfiguration struct {
	SerializerManager   *serializer.SerializerManager
	RT                  http.RoundTripper
	RemoteServers       []*url.URL
	DishCachePath       string
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

	// 应该序列化到磁盘的时候使用
	serializerManager := serializer.NewSerializerManager()
	// 获取 roundTripper 表示执行单个HTTP事务的能力，获得给定请求的响应
	rt, err := prepareRoundTripper()
	if err != nil {
		return nil, fmt.Errorf("could not new round tripper, %w", err)
	}

	cfg := &EdgeProxyConfiguration{
		RemoteServers:       us,
		BindAddr:            net.JoinHostPort("127.0.0.1", "10267"),
		EdgeProxyServerAddr: net.JoinHostPort("127.0.0.1", "10261"),
		DishCachePath:       options.DiskCachePath,
		SerializerManager:   serializerManager,
		RT:                  rt,
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

// todo: 传入 flag 表示是否是测试，否则只能 inClusterConfig
func prepareRoundTripper() (http.RoundTripper, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return rest.TransportFor(cfg)
}
