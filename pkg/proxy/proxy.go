package proxy

import (
	"fmt"
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
)

type HandlerFactory interface {
	Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error)
}

var (
	proxyHandlerFactories = []HandlerFactory{}
)

func Register(factory HandlerFactory) {
	proxyHandlerFactories = append(proxyHandlerFactories, factory)
}

func GetProxyHandler(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {
	if len(proxyHandlerFactories) != 1 {
		return nil, fmt.Errorf("no handler factory is prepared")
	}

	return proxyHandlerFactories[0].Init(cfg, stopCh)
}
