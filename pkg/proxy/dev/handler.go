package dev

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/openyurtio/openyurt/pkg/yurthub/storage/factory"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/types"

	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
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
	localProxy   LoadBalancer
	cc           *cacheChecker
	cfg          *config.EdgeProxyConfiguration
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if d.loadBalancer.IsHealthy() {
		d.loadBalancer.ServeHTTP(rw, req)
		return
	}

	// if lb not healthy, then use localProxy
	d.localProxy.ServeHTTP(rw, req)
}

func (d *devFactory) initCacheMgr() {
	storageManager, err := factory.CreateStorage(d.cfg.DiskCachePath)
	if err != nil {
		klog.Errorf("could not create storage manager, %v", err)
		return
	}
	cacheMgr := NewCacheMgr(storageManager)
	d.loadBalancer.SetCacheMgr(cacheMgr)
	d.localProxy.SetCacheMgr(cacheMgr)
	klog.Infof("set cache mgr")
}

func (d *devFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {

	d.cfg = cfg

	serverCfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	resolver := server.NewRequestInfoResolver(serverCfg)
	d.resolver = resolver

	cc := NewCacheChecker()
	d.cc = cc

	remoteServer := cfg.RemoteServers[0] // 假设一定成立

	//cacheMgr := NewCacheMgr(cfg.StorageMgr)
	lb, _ := NewRemoteProxy(remoteServer, nil, cfg.RT, cc, stopCh)
	d.loadBalancer = lb

	// local proxy when lb is not healthy
	d.localProxy = NewLocalProxy(nil, lb.IsHealthy)

	return d.buildHandlerChain(d), nil
}

// 增加中间件
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	//handler = yurthubutil.WithRequestContentType(handler)
	handler = d.printCreateReqBody(handler)
	handler = d.countReq(handler)
	//handler = d.WithMaxInFlightLimit(handler, 200) // 两百个并发

	// inject request info
	handler = filters.WithRequestInfo(handler, d.resolver)

	return handler
}

func (d *devFactory) printCreateReqBody(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if info, ok := apirequest.RequestInfoFrom(ctx); ok {
			// 打印 create 的 body 判断 resourceusage 的创建 body，然后本地进行 benchmark
			if info.Verb == "create" && !d.cc.CanCache() {
				pr, prc := util.NewDualReadCloser(req, req.Body, false)
				go func(reader io.ReadCloser) {
					//reqBody, err := io.ReadAll(prc)
					var createReq types.ResourceCreateReq
					err := json.NewDecoder(prc).Decode(&createReq)
					if err != nil {
						klog.Errorf("readAll req.Body err: %v", err)
						return
					}
					//klog.Infof("info: %v, req.Body is %v", util.ReqString(req), string(reqBody))
					if createReq.Metadata.Labels.Type == "consistency" { // set cache true
						klog.Infof("set cache true")
						d.cc.SetCanCache()
						d.initCacheMgr()
					}
				}(prc)
				req.Body = pr
				// 错误使用方法 req.Body 读完一次就没了，所以不能直接读完就不管
				//b, err := io.ReadAll(req.Body)
				//if err != nil {
				//	klog.Errorf("read req body err: %v", err)
				//	return
				//}
				//klog.Infof("req.Body is %v", string(b))
			}
		}

		handler.ServeHTTP(rw, req)
	})
}

func (d *devFactory) countReq(handler http.Handler) http.Handler {
	var count = 0
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
		if strings.Contains(labelSelector, resourceLabel) {
			count++
			klog.Infof("resource usage count is %v", count)
		}

		handler.ServeHTTP(w, req)
	})
}

// WithMaxInFlightLimit limits the number of in-flight requests. and when in flight
// requests exceeds the threshold, the following incoming requests will be rejected.
func (d *devFactory) WithMaxInFlightLimit(handler http.Handler, limit int) http.Handler {
	// 闭包，所以这里不会重复初始化
	var reqChan chan bool
	if limit > 0 {
		reqChan = make(chan bool, limit)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		select {
		case reqChan <- true:
			klog.Infof("start proxying: %s %s, in flight requests: %d", strings.ToLower(req.Method), req.URL.String(), len(reqChan))
			defer func() {
				<-reqChan
				klog.Infof("%s request completed, left %d requests in flight", req.URL.String(), len(reqChan))
			}()
			handler.ServeHTTP(w, req)
		default:
			// Return a 429 status indicating "Too Many Requests"
			klog.Errorf("Too many requests, please try again later, %s", req.URL.String())
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
	})
}
