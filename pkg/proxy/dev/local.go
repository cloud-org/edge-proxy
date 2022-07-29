package dev

import (
	"fmt"
	"net/http"

	"github.com/openyurtio/openyurt/pkg/yurthub/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// IsHealthy is func for fetching healthy status of remote server
type IsHealthy func() bool

// LocalProxy is responsible for handling requests when remote servers are unhealthy
type LocalProxy struct {
	cacheMgr  *CacheMgr
	isHealthy IsHealthy
}

// NewLocalProxy creates a *LocalProxy
func NewLocalProxy(cacheMgr *CacheMgr, isHealthy IsHealthy) *LocalProxy {
	return &LocalProxy{
		cacheMgr:  cacheMgr,
		isHealthy: isHealthy,
	}
}

// ServeHTTP implements http.Handler for LocalProxy
func (lp *LocalProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	ctx := req.Context()
	if reqInfo, ok := apirequest.RequestInfoFrom(ctx); ok && reqInfo != nil && reqInfo.IsResourceRequest {
		switch reqInfo.Verb {
		case "delete", "deletecollection":
			err = localDelete(w, req)
		default: // list., get, update
			err = lp.localReqCache(w, req)
		}

		if err != nil {
			klog.Errorf("could not proxy local for %s, %v", util.ReqString(req), err)
			util.Err(err, w, req)
		}
	} else {
		klog.Errorf("request(%s) is not supported when cluster is unhealthy", util.ReqString(req))
		util.Err(apierrors.NewBadRequest(fmt.Sprintf("request(%s) is not supported when cluster is unhealthy", util.ReqString(req))), w, req)
	}
}

// localDelete handles Delete requests when remote servers are unhealthy
func localDelete(w http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	info, _ := apirequest.RequestInfoFrom(ctx)
	s := &metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusForbidden,
		Reason: metav1.StatusReasonForbidden,
		Details: &metav1.StatusDetails{
			Name:  info.Name,
			Group: info.Namespace,
			Kind:  info.Resource,
		},
		Message: "delete request is not supported in local cache",
	}

	util.WriteObject(http.StatusForbidden, s, w, req)
	return nil
}

// localReqCache handles Get/List/Update requests when remote servers are unhealthy
func (lp *LocalProxy) localReqCache(w http.ResponseWriter, req *http.Request) error {
	klog.Infof("now req cache...")

	// filter consistency
	info, _ := apirequest.RequestInfoFrom(req.Context())

	labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
	if !checkLabel(info, labelSelector, consistencyLabel) {
		return fmt.Errorf("not list consistency label")
	}

	obj, err := lp.cacheMgr.QueryCache(info)
	if err != nil {
		klog.Errorf("查询缓存失败 err: %v", err)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(obj)
	if err != nil {
		klog.Errorf("rw.Write err: %v", err)
		return err
	}

	return nil
}
