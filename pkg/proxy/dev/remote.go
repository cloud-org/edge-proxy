package dev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"

	"github.com/openyurtio/openyurt/pkg/yurthub/kubernetes/serializer"
	"k8s.io/client-go/kubernetes"

	yurthubutil "github.com/openyurtio/openyurt/pkg/yurthub/util"
	"k8s.io/apimachinery/pkg/util/httpstream"
	proxy2 "k8s.io/apimachinery/pkg/util/proxy"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("failed while proxying request %s, %v", req.URL, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type RemoteProxy struct {
	remoteServer        *url.URL
	reverseProxy        *httputil.ReverseProxy
	currentTransport    http.RoundTripper
	upgradeAwareHandler *proxy2.UpgradeAwareHandler
	serializerManager   *serializer.SerializerManager
	cacheMgr            cachemanager.CacheManager
	stopCh              <-chan struct{}
	checker             *checker
}

// NewRemoteProxy 参数之后接着补充
func NewRemoteProxy(
	remoteServer *url.URL,
	cacheMgr cachemanager.CacheManager,
	transport http.RoundTripper,
	serializerManager *serializer.SerializerManager,
	client *kubernetes.Clientset,
	stopCh <-chan struct{},
) (*RemoteProxy, error) {

	rproxy := &RemoteProxy{
		remoteServer:      remoteServer,
		currentTransport:  transport,
		serializerManager: serializerManager,
		cacheMgr:          cacheMgr,
		stopCh:            stopCh,
	}

	rproxy.checker = NewChecker(remoteServer, client)
	rproxy.checker.start(rproxy.stopCh) // start checker

	// todo: websocket 处理 可以实际测试下
	upgradeAwareHandler := proxy2.NewUpgradeAwareHandler(
		remoteServer,
		rproxy.currentTransport,
		false,
		true,
		&responder{},
	)
	upgradeAwareHandler.UseRequestLocation = true

	rproxy.upgradeAwareHandler = upgradeAwareHandler

	// 初始化反向代理
	rproxy.reverseProxy = httputil.NewSingleHostReverseProxy(rproxy.remoteServer)

	rproxy.reverseProxy.Transport = rproxy // 注入自定义的 transport
	rproxy.reverseProxy.FlushInterval = -1
	rproxy.reverseProxy.ModifyResponse = rproxy.modifyResponse
	rproxy.reverseProxy.ErrorHandler = rproxy.errorHandler

	return rproxy, nil
}

func (rp *RemoteProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if httpstream.IsUpgradeRequest(req) {
		klog.Infof("get upgrade request %s", req.URL)
		rp.upgradeAwareHandler.ServeHTTP(rw, req)
		return
	}

	rp.reverseProxy.ServeHTTP(rw, req)
}

func (rp *RemoteProxy) RoundTrip(request *http.Request) (*http.Response, error) {
	// http.RoundTripper
	return rp.currentTransport.RoundTrip(request)
}

func (rp *RemoteProxy) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("remote proxy error handler: %s, %v", req.URL.String(), err)
	rw.WriteHeader(http.StatusBadGateway)
	// todo: 从缓存中进行查询 一致性查询中使用
}

func (rp *RemoteProxy) Name() string {
	return rp.remoteServer.String()
}

func (rp *RemoteProxy) IsHealthy() bool {
	return rp.checker.isHealthy()
}

func (rp *RemoteProxy) modifyResponse(resp *http.Response) error {
	// 过滤和设置缓存
	if resp == nil || resp.Request == nil {
		klog.Infof("no request info in response, skip cache response")
		return nil
	}

	req := resp.Request
	ctx := req.Context()

	labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
	// if test resource then return directly
	if strings.Contains(labelSelector, resourceLabel) {
		return nil
	}

	// re-added transfer-encoding=chunked response header for watch request
	info, exists := apirequest.RequestInfoFrom(ctx)
	if exists {
		if info.Verb == "watch" {
			klog.V(5).Infof(
				"add transfer-encoding=chunked header into response for req %s",
				yurthubutil.ReqString(req),
			)
			h := resp.Header
			if hv := h.Get("Transfer-Encoding"); hv == "" {
				h.Add("Transfer-Encoding", "chunked")
			}
		}
	}

	// 成功响应
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusPartialContent {
		// prepare response content type
		reqContentType, _ := yurthubutil.ReqContentTypeFrom(ctx)
		respContentType := resp.Header.Get("Content-Type")
		if len(respContentType) == 0 {
			respContentType = reqContentType
		}
		ctx = yurthubutil.WithRespContentType(ctx, respContentType)
		req = req.WithContext(ctx)

		klog.Infof("request info is %+v\n", info)
		// filter response data
		if checkLabel(info, labelSelector, filterLabel) {
			wrapBody, needUncompressed := yurthubutil.NewGZipReaderCloser(resp.Header, resp.Body, req, "filter")
			s := CreateSerializer(req, rp.serializerManager)
			if s == nil {
				klog.Errorf("CreateSerializer is nil")
				return nil
			}
			filterManager := NewSkipListFilter(info.Resource, s, "skip-")

			size, filterRc, err := NewFilterReadCloser(wrapBody, filterManager)
			if err != nil {
				klog.Errorf("failed to filter response for %s, %v", yurthubutil.ReqString(req), err)
				return err
			}
			resp.Body = filterRc
			if size > 0 {
				resp.ContentLength = int64(size)
				resp.Header.Set("Content-Length", fmt.Sprint(size))
			}

			// after gunzip in filter, the header content encoding should be removed.
			// because there's no need to gunzip response.body again.
			if needUncompressed {
				resp.Header.Del("Content-Encoding")
			}
		}

		if checkLabel(info, labelSelector, filterLabel) || checkLabel(info, labelSelector, funcLabel) {
			klog.Infof("func/filter not need to cache")
			return nil
		}

		// cache
		if (info.IsResourceRequest && info.Verb == "list" &&
			(info.Resource == "pods" || info.Resource == "configmaps") && labelSelector == "") ||
			checkLabel(info, labelSelector, consistencyLabel) {
			// cache resp with storage interface
			if rp.cacheMgr != nil {
				rc, prc := yurthubutil.NewDualReadCloser(req, resp.Body, true)
				wrapPrc, _ := yurthubutil.NewGZipReaderCloser(resp.Header, prc, req, "cache-manager")
				go func(req *http.Request, prc io.ReadCloser, stopCh <-chan struct{}) {
					klog.Infof("cache consistency response")
					err := rp.cacheMgr.CacheResponse(req, prc, stopCh)
					if err != nil && err != io.EOF && !errors.Is(err, context.Canceled) {
						klog.Errorf("%s response cache ended with error, %v", yurthubutil.ReqString(req), err)
					}
				}(req, wrapPrc, rp.stopCh)

				resp.Body = rc
			}
		}

	}

	return nil
}

func CreateSerializer(req *http.Request, sm *serializer.SerializerManager) *serializer.Serializer {
	ctx := req.Context()
	respContentType, _ := yurthubutil.RespContentTypeFrom(ctx)
	info, _ := apirequest.RequestInfoFrom(ctx)
	if respContentType == "" || info == nil || info.APIVersion == "" || info.Resource == "" {
		klog.Infof("CreateSerializer failed , info is :%+v", info)
		return nil
	}
	return sm.CreateSerializer(respContentType, info.APIGroup, info.APIVersion, info.Resource)
}

func checkLabel(info *apirequest.RequestInfo, selector string, label string) bool {
	if info.IsResourceRequest && info.Verb == "list" &&
		(info.Resource == "pods" || info.Resource == "configmaps") &&
		strings.Contains(selector, label) { // only for consistency
		return true
	}

	return false
}
