package dev

import (
	"fmt"
	"io"
	"net/http"

	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	"github.com/openyurtio/openyurt/pkg/yurthub/util"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	yurthubutil "github.com/openyurtio/openyurt/pkg/yurthub/proxy/util"
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
	localProxy   http.Handler
}

func (d *devFactory) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if d.loadBalancer.IsHealthy() {
		d.loadBalancer.ServeHTTP(rw, req)
		return
	}

	// if lb not healthy, then use localProxy
	d.localProxy.ServeHTTP(rw, req)
}

func (d *devFactory) Init(cfg *config.EdgeProxyConfiguration, stopCh <-chan struct{}) (http.Handler, error) {

	serverCfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	resolver := server.NewRequestInfoResolver(serverCfg)
	d.resolver = resolver

	klog.Infof("new cache manager with storage wrapper and serializer manager")
	// sharedFactory temporarily set as nil
	cacheMgr, err := cachemanager.NewCacheManager(
		cfg.StorageWrapper,
		cfg.SerializerManager,
		cfg.RESTMapperManager,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not new cache manager, %w", err)
	}

	remoteServer := cfg.RemoteServers[0] // 假设一定成立
	lb, _ := NewRemoteProxy(remoteServer, cacheMgr, cfg.RT, cfg.SerializerManager, cfg.Client, stopCh)
	d.loadBalancer = lb

	// local proxy when lb is not healthy
	d.localProxy = NewLocalProxy(cacheMgr, lb.IsHealthy)

	return d.buildHandlerChain(d), nil
}

// 增加中间件
func (d *devFactory) buildHandlerChain(handler http.Handler) http.Handler {
	handler = yurthubutil.WithRequestContentType(handler)
	handler = WithCacheHeaderCheck(handler)
	//handler = WithListRequestSelector(handler)
	handler = printCreateReqBody(handler)

	// inject request info
	handler = filters.WithRequestInfo(handler, d.resolver)

	return handler
}

func WithCacheHeaderCheck(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if info, ok := apirequest.RequestInfoFrom(ctx); ok {
			labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
			if (info.IsResourceRequest && info.Verb == "list" &&
				(info.Resource == "pods" || info.Resource == "configmaps") && labelSelector == "") ||
				checkLabel(info, labelSelector, consistencyLabel) {
				klog.Infof("req labelSelector is %v, add cache header and comp", labelSelector)
				// add cache header
				ctx = util.WithReqCanCache(ctx, true)
				// add comp bench
				ctx = util.WithClientComponent(ctx, "bench")
				req = req.WithContext(ctx)
			}
		}

		handler.ServeHTTP(w, req)
	})
}

func printCreateReqBody(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if info, ok := apirequest.RequestInfoFrom(ctx); ok {
			// 打印 create 的 body 判断 resourceusage 的创建 body，然后本地进行 benchmark
			if info.Verb == "create" {
				pr, prc := util.NewDualReadCloser(req, req.Body, false)
				go func(reader io.ReadCloser) {
					reqBody, err := io.ReadAll(prc)
					if err != nil {
						klog.Errorf("readAll req.Body err: %v", err)
						return
					}
					klog.Infof("info: %v, req.Body is %v", util.ReqString(req), string(reqBody))
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
