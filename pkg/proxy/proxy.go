package proxy

import (
	"fmt"
	"k8s.io/klog/v2"
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
)

var (
	proxyHandlerFactories = map[string]HandlerFactory{}
)

func Register(name string, factory HandlerFactory) {
	if _, ok := proxyHandlerFactories[name]; ok {
		klog.Infof("proxy handler %s already has been registered, name")
		return
	}

	proxyHandlerFactories[name] = factory
}

func GetProxyHandler(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {
	if cfg.EnableSampleHandler {
		return proxyHandlerFactories["sample"].Init(cfg, stopCh)
	}

	for name, factory := range proxyHandlerFactories {
		klog.Infof("get a proxy handler from %s", name)
		return factory.Init(cfg, stopCh)
	}

	return nil, fmt.Errorf("no proxy handler is prepared")
}
