package dev

import (
	"fmt"
	"net/http"

	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	"github.com/openyurtio/openyurt/pkg/yurthub/proxy/local"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	yurthubutil "github.com/openyurtio/openyurt/pkg/yurthub/proxy/util"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
)

func init() {
	proxy.Register("sample", &devFactory{})
}

type devFactory struct {
	resolver     apirequest.RequestInfoResolver
	loadBalancer LoadBalancer
	localProxy   http.Handler
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if d.loadBalancer.IsHealthy() {
		d.loadBalancer.ServeHTTP(rw, req)
		return
	}

	// if lb not healthy, then use localProxy
	d.localProxy.ServeHTTP(rw, req)
}

func (d *devFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {

	serverCfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	resolver := server.NewRequestInfoResolver(serverCfg)
	d.resolver = resolver

	remoteServer := cfg.RemoteServers[0] // 假设一定成立
	lb, _ := NewRemoteProxy(remoteServer, cfg.RT, cfg.SerializerManager, cfg.Client, stopCh)
	d.loadBalancer = lb

	klog.Infof("new cache manager with storage wrapper and serializer manager")
	// sharedFactory temporarily set as nil
	cacheMgr, err := cachemanager.NewCacheManager(
		cfg.StorageWrapper,
		cfg.SerializerManager,
		cfg.RESTMapperManager,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not new cache manager, %w", err)
	}

	// local proxy when lb is not healthy
	d.localProxy = local.NewLocalProxy(cacheMgr, lb.IsHealthy)

	return d.buildHandlerChain(d), nil
}

// 增加中间件
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	handler = yurthubutil.WithRequestContentType(handler)
	handler = yurthubutil.WithRequestClientComponent(handler)
	handler = filters.WithRequestInfo(handler, d.resolver)

	return handler
}
