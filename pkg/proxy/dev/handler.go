package dev

import (
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	yurthubutil "github.com/openyurtio/openyurt/pkg/yurthub/proxy/util"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
)

func init() {
	proxy.Register("dev", &devFactory{})
}

type devFactory struct {
	resolver     apirequest.RequestInfoResolver
	loadBalancer LoadBalancer
	localProxy   http.Handler
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// todo: 健康检查 unhealthy 进入 localProxy
	d.loadBalancer.ServeHTTP(rw, req)
}

func (d *devFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {

	serverCfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	resolver := server.NewRequestInfoResolver(serverCfg)
	d.resolver = resolver

	remoteServer := cfg.RemoteServers[0] // 假设一定成立
	serializerManager := serializer.NewSerializerManager()
	lb, _ := NewRemoteProxy(remoteServer, cfg.RT, serializerManager)
	d.loadBalancer = lb

	return d.buildHandlerChain(d), nil
}

// 增加中间件
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	handler = yurthubutil.WithRequestContentType(handler)
	handler = yurthubutil.WithRequestClientComponent(handler)
	handler = filters.WithRequestInfo(handler, d.resolver)

	return handler
}
