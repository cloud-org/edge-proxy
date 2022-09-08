package dev

import (
	"net/http"
	"strings"

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
	// resourceCache if resource has cached or not
	resourceCache bool
	// resourceNs resource cache namespace
	resourceNs string
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
	handler = d.returnCacheResourceUsage(handler)

	return handler
}

//returnCacheResourceUsage if labelSelector contains type=resourceusage, then return mem data if ok
func (d *devFactory) returnCacheResourceUsage(handler http.Handler) http.Handler {
	var count int

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			handler.ServeHTTP(rw, req)
			return
		}
		// if resource usage cache, then return, else continue
		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
		if d.resourceCache && strings.Contains(labelSelector, resourceLabel) {
			//klog.Infof("return resource cache")
			count++
			klog.V(5).Infof("resource usage count is %v", count)
			res, ok := d.cacheMgr.QueryCacheMem("configmaps", d.resourceNs, resourceType)
			if !ok {
				klog.Errorf("may be not resource cache")
				goto end
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_, err := rw.Write(res)
			if err != nil {
				klog.Errorf("rw.Write err: %v", err)
				goto end
			}
			// return if not err
			return
		}
	end:

		info, err := d.resolver.NewRequestInfo(req)
		if err != nil {
			klog.Errorf("resolver request info err: %v", err)
			return
		}
		// inject info
		req = req.WithContext(apirequest.WithRequestInfo(req.Context(), info))
		// no resource cache
		handler.ServeHTTP(rw, req)
		if checkLabel(info, labelSelector, resourceLabel) {
			d.resourceCache = true        // set cache true
			d.resourceNs = info.Namespace // set ns
			count++
			klog.V(5).Infof("first resource usage count is %v", count)
		}
	})
}
