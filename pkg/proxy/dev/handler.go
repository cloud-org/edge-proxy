package dev

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	"github.com/openyurtio/openyurt/pkg/yurthub/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/edge-proxy/app/config"
	"code.aliyun.com/openyurt/edge-proxy/pkg/proxy"
	yurthubutil "github.com/openyurtio/openyurt/pkg/yurthub/proxy/util"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
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

func WithListRequestSelector(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if info, ok := apirequest.RequestInfoFrom(ctx); ok {
			if info.IsResourceRequest && info.Verb == "list" && info.Name == "" {
				// list request with fieldSelector=metadata.name does not need to set selector string
				opts := metainternalversion.ListOptions{}
				if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, &opts); err == nil {
					str := selectorString(opts.LabelSelector, opts.FieldSelector)
					//klog.Infof("ls and fs str is %v", str)
					if str != "" {
						ctx = util.WithListSelector(ctx, str)
						req = req.WithContext(ctx)
					}
				}
			}
		}

		handler.ServeHTTP(w, req)
	})
}

// selectorString returns the string of label and field selector
func selectorString(lSelector labels.Selector, fSelector fields.Selector) string {
	var ls string
	var fs string
	if lSelector != nil {
		ls = lSelector.String()
	}

	if fSelector != nil {
		fs = fSelector.String()
	}

	switch {
	case ls != "" && fs != "":
		return strings.Join([]string{ls, fs}, "&")

	case ls != "":
		return ls

	case fs != "":
		return fs
	}

	return ""
}
