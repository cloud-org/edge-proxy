package proxy

import (
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
)

type HandlerFactory interface {
	Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error)
}
