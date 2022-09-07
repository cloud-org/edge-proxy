package dev

import (
	"net/http"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util"

	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	"k8s.io/apimachinery/pkg/util/sets"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
)

func init() {
	proxy.Register("sample", &devFactory{})
}

type devFactory struct {
	// for inject request info to http.Request
	resolver apirequest.RequestInfoResolver
	// remoteProxy reverseProxy for remote server
	remoteProxy APIServerProxy
	// localProxy use local proxy when remote server unhealthy
	localProxy APIServerProxy
	cfg        *config.EdgeProxyConfiguration
	// cacheMgr cache manager
	cacheMgr *CacheMgr
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if d.remoteProxy.IsHealthy() {
		d.remoteProxy.ServeHTTP(rw, req)
		return
	}

	// if remoteProxy not healthy, then use localProxy
	d.localProxy.ServeHTTP(rw, req)
}

//initCacheMgr init cache mgr
func (d *devFactory) initCacheMgr() (*CacheMgr, error) {
	storageManager, err := util.NewDiskStorage(d.cfg.DiskCachePath)
	if err != nil {
		klog.Errorf("could not create storage manager, %v", err)
		return nil, err
	}
	cacheMgr := NewCacheMgr(storageManager)
	return cacheMgr, nil
}

func (d *devFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {

	d.cfg = cfg

	serverCfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	resolver := server.NewRequestInfoResolver(serverCfg)
	d.resolver = resolver

	remoteServer := cfg.RemoteServers[0] // 假设一定成立

	cacheMgr, err := d.initCacheMgr()
	if err != nil {
		return nil, err
	}

	d.cacheMgr = cacheMgr
	// init remoteProxy
	lb, _ := NewRemoteProxy(remoteServer, cacheMgr, cfg.RT, stopCh)
	d.remoteProxy = lb

	// init localProxy
	d.localProxy = NewLocalProxy(cacheMgr, lb.IsHealthy)

	return d.buildHandlerChain(d), nil
}

// buildHandlerChain use middleware for handler
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	handler = d.returnLabelSelectorList(handler)

	return handler
}

//returnLabelSelectorList if labelSelector list, then return mem data if ok
func (d *devFactory) returnLabelSelectorList(handler http.Handler) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// only handle HTTP GET method
		if req.Method != http.MethodGet {
			handler.ServeHTTP(rw, req)
			return
		}

		info, err := d.resolver.NewRequestInfo(req)
		if err != nil {
			klog.Errorf("resolver request info err: %v", err)
			return
		}

		// inject request info
		req = req.WithContext(apirequest.WithRequestInfo(req.Context(), info))

		// only for info.Verb equal list
		if info.Verb != "list" {
			handler.ServeHTTP(rw, req)
			return
		}

		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter

		// query from cachemgr
		res, ok := d.cacheMgr.QueryCacheMem(info.Resource, info.Namespace, labelSelector)
		if !ok {
			klog.Errorf("may be not resource cache")
			handler.ServeHTTP(rw, req)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		// write cache result
		_, err = rw.Write(res)
		if err != nil {
			klog.Errorf("rw.Write err: %v", err)
			handler.ServeHTTP(rw, req)
			return
		}
		// return if not err
		return
	})
}
