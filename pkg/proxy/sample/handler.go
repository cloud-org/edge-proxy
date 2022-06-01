package sample

import (
	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
)

func init() {
	proxy.Register(&sampleFactory{})
}

type sampleFactory struct{}

func (sf *sampleFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {
	// simple example: use httputil.ReverseProxy to proxy requests.

	reverseProxy := httputil.NewSingleHostReverseProxy(cfg.RemoteServers[0])
	reverseProxy.Transport = cfg.RT
	reverseProxy.FlushInterval = -1
	reverseProxy.ErrorHandler = errorHandler

	return reverseProxy, nil
}

func errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("remote proxy error handler: %s, %v", req.URL.String(), err)
	rw.WriteHeader(http.StatusBadGateway)
}
