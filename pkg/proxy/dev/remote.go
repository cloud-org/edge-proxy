package dev

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"
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
}

// NewRemoteProxy 参数之后接着补充
func NewRemoteProxy(
	remoteServer *url.URL,
	transport http.RoundTripper,
	serializerManager *serializer.SerializerManager,
) (*RemoteProxy, error) {

	rproxy := &RemoteProxy{
		remoteServer:      remoteServer,
		currentTransport:  transport,
		serializerManager: serializerManager,
	}

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
	// todo: 健康检查
	return true
}

func (rp *RemoteProxy) modifyResponse(resp *http.Response) error {
	// 过滤和设置缓存
	if resp == nil || resp.Request == nil {
		klog.Infof("no request info in response, skip cache response")
		return nil
	}

	req := resp.Request
	ctx := req.Context()

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
		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
		if info.IsResourceRequest && info.Verb == "list" &&
			(info.Resource == "pods" || info.Resource == "configmaps") &&
			strings.Contains(labelSelector, "type=filter") {
			wrapBody, needUncompressed := yurthubutil.NewGZipReaderCloser(resp.Header, resp.Body, req, "filter")
			s := CreateSerializer(req, rp.serializerManager)
			if s == nil {
				klog.Errorf("CreateSerializer is nil")
				return nil
			}
			filterManager := NewSkipListFilter(info.Resource, s)

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

		// todo: cache 做缓存方便后续查询
		//if info.Verb == "create" {
		//
		//}

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
