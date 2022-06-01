package sample

import (
	"net/http"
	"net/http/httputil"

	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
)

func init() {
	proxy.Register("sample", &sampleFactory{})
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
