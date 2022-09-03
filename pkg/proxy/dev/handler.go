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
	resolver     apirequest.RequestInfoResolver
	loadBalancer LoadBalancer
	localProxy   LoadBalancer
	cfg          *config.EdgeProxyConfiguration
	cacheMgr     *CacheMgr
	//resourceCache bool
	//resourceNs    string
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if d.loadBalancer.IsHealthy() {
		d.loadBalancer.ServeHTTP(rw, req)
		return
	}

	// if lb not healthy, then use localProxy
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
	lb, _ := NewRemoteProxy(remoteServer, cacheMgr, cfg.RT, stopCh)
	d.loadBalancer = lb

	// local proxy when lb is not healthy
	d.localProxy = NewLocalProxy(cacheMgr, lb.IsHealthy)

	return d.buildHandlerChain(d), nil
}

// buildHandlerChain use middleware
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	//handler = yurthubutil.WithRequestContentType(handler)
	//handler = d.printCreateReqBody(handler)
	handler = d.returnCacheResourceUsage(handler)
	//handler = d.countReq(handler)
	//handler = d.WithMaxInFlightLimit(handler, 200) // 两百个并发

	// inject request info
	//handler = filters.WithRequestInfo(handler, d.resolver)

	return handler
}

//returnCacheResourceUsage if labelSelector contains type=resourceusage, then return mem data if ok
func (d *devFactory) returnCacheResourceUsage(handler http.Handler) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			handler.ServeHTTP(rw, req)
			return
		}

		info, err := d.resolver.NewRequestInfo(req)
		if err != nil {
			klog.Errorf("resolver request info err: %v", err)
			return
		}

		// inject info
		req = req.WithContext(apirequest.WithRequestInfo(req.Context(), info))

		if info.Verb != "list" {
			handler.ServeHTTP(rw, req)
			return
		}

		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter

		res, ok := d.cacheMgr.QueryCacheMem(info.Resource, info.Namespace, labelSelector)
		if !ok {
			klog.Errorf("may be not resource cache")
			handler.ServeHTTP(rw, req)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(res)
		if err != nil {
			klog.Errorf("rw.Write err: %v", err)
		}
		// return if not err
		return

	})
}
