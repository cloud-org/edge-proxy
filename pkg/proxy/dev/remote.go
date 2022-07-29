package dev

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

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
	cacheMgr            *CacheMgr
	stopCh              <-chan struct{}
	checker             *checker
	cc                  *cacheChecker
}

// NewRemoteProxy 参数之后接着补充
func NewRemoteProxy(
	remoteServer *url.URL,
	cacheMgr *CacheMgr,
	transport http.RoundTripper,
	cc *cacheChecker,
	stopCh <-chan struct{},
) (*RemoteProxy, error) {

	rproxy := &RemoteProxy{
		remoteServer:     remoteServer,
		currentTransport: transport,
		cacheMgr:         cacheMgr,
		cc:               cc,
		stopCh:           stopCh,
	}

	rproxy.checker = NewChecker(remoteServer)
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
	// todo: remove
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
			h := resp.Header
			if hv := h.Get("Transfer-Encoding"); hv == "" {
				h.Add("Transfer-Encoding", "chunked")
				klog.Infof("add Transfer-Encoding header")
			}
		}
	}

	// 成功响应
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusPartialContent {
		klog.Infof("request info is %+v\n", info)
		// filter response data
		if checkLabel(info, labelSelector, filterLabel) {
			// todo: 重写 gzip reader 因为里面有对 component 进行获取
			wrapBody, needUncompressed := yurthubutil.NewGZipReaderCloser(resp.Header, resp.Body, req, "filter")
			size, filterRc, err := NewFilterReadCloser(wrapBody, info.Resource, "skip-")
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
			klog.Infof("functional/filter label not need to cache")
			return nil
		}

		// cache
		if (info.IsResourceRequest && info.Verb == "list" &&
			(info.Resource == "pods" || info.Resource == "configmaps") && labelSelector == "") ||
			checkLabel(info, labelSelector, consistencyLabel) {
			// cache resp with storage interface
			if rp.cacheMgr != nil && rp.cc.CanCache() {
				rc, prc := yurthubutil.NewDualReadCloser(req, resp.Body, true)
				wrapPrc, _ := yurthubutil.NewGZipReaderCloser(resp.Header, prc, req, "cache-manager")
				go func(req *http.Request, prc io.ReadCloser) {
					klog.Infof("cache consistency response")
					err := rp.cacheMgr.CacheResponse(info, prc)
					if err != nil {
						klog.Errorf("%s response cache ended with error, %v", info.Resource, err)
					}
				}(req, wrapPrc)

				resp.Body = rc
			}
		}

	}

	return nil
}

func checkLabel(info *apirequest.RequestInfo, selector string, label string) bool {
	if info.IsResourceRequest && info.Verb == "list" &&
		(info.Resource == "pods" || info.Resource == "configmaps") &&
		strings.Contains(selector, label) { // only for consistency
		return true
	}

	return false
}
