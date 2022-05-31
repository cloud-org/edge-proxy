package proxy

import (
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
)

// NewEdgeReverseProxyHandler creates a http handler for proxying all of incoming requests.
func NewEdgeReverseProxyHandler(cfg *config.EdgeProxyConfiguration, rt http.RoundTripper, stopCh <-chan struct{}) (http.Handler, error) {
	// simple example: use httputil.ReverseProxy to proxy requests.

	//reverseProxy := httputil.NewSingleHostReverseProxy(cfg.RemoteServers[0])
	//reverseProxy.Transport = rt
	//reverseProxy.FlushInterval = -1
	//reverseProxy.ErrorHandler = errorHandler
	//
	//return reverseProxy, nil

	return nil, nil
}

//func errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
//	klog.Errorf("remote proxy error handler: %s, %v", req.URL.String(), err)
//	rw.WriteHeader(http.StatusBadGateway)
//}
