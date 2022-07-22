package dev

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	hubmeta "github.com/openyurtio/openyurt/pkg/yurthub/kubernetes/meta"
	"github.com/openyurtio/openyurt/pkg/yurthub/storage"
	"github.com/openyurtio/openyurt/pkg/yurthub/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// IsHealthy is func for fetching healthy status of remote server
type IsHealthy func() bool

// LocalProxy is responsible for handling requests when remote servers are unhealthy
type LocalProxy struct {
	cacheMgr  cachemanager.CacheManager
	isHealthy IsHealthy
}

// NewLocalProxy creates a *LocalProxy
func NewLocalProxy(cacheMgr cachemanager.CacheManager, isHealthy IsHealthy) *LocalProxy {
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
		klog.V(3).Infof("go into local proxy for request %s", util.ReqString(req))
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
	//if !lp.cacheMgr.CanCacheFor(req) {
	//	klog.Errorf("can not cache for %s", util.ReqString(req))
	//	return apierrors.NewBadRequest(fmt.Sprintf("can not cache for %s", util.ReqString(req)))
	//}

	obj, err := lp.cacheMgr.QueryCache(req)
	// filter consistency
	info, _ := apirequest.RequestInfoFrom(req.Context())

	labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
	if !checkLabel(info, labelSelector, consistencyLabel) {
		return fmt.Errorf("not list consistency label")
	}

	switch info.Resource {
	case "pods":
		podList, ok := obj.(*v1.PodList)
		if !ok {
			klog.Errorf("req *v1.PodList 断言失败")
			goto end
		}
		var items []v1.Pod
		for i := 0; i < len(podList.Items); i++ {
			if podList.Items[i].Labels["type"] == "consistency" {
				klog.Infof("add item %s", podList.Items[i].Name)
				items = append(items, podList.Items[i])
			}
		}
		podList.Items = items
		obj = podList // re
	case "configmaps":
		configMapList, ok := obj.(*v1.ConfigMapList)
		if !ok {
			klog.Errorf("req *v1.ConfigMapList 断言失败")
			goto end
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configMapList.Items); i++ {
			if configMapList.Items[i].Labels["type"] == "consistency" {
				klog.Infof("add item %s", configMapList.Items[i].Name)
				items = append(items, configMapList.Items[i])
			}
		}
		configMapList.Items = items
		obj = configMapList // re
	}

end:

	if errors.Is(err, storage.ErrStorageNotFound) || errors.Is(err, hubmeta.ErrGVRNotRecognized) {
		klog.Errorf("object not found for %s", util.ReqString(req))
		reqInfo, _ := apirequest.RequestInfoFrom(req.Context())
		return apierrors.NewNotFound(schema.GroupResource{Group: reqInfo.APIGroup, Resource: reqInfo.Resource}, reqInfo.Name)
	} else if err != nil {
		klog.Errorf("failed to query cache for %s, %v", util.ReqString(req), err)
		return apierrors.NewInternalError(err)
	} else if obj == nil {
		klog.Errorf("no cache object for %s", util.ReqString(req))
		return apierrors.NewInternalError(fmt.Errorf("no cache object for %s", util.ReqString(req)))
	}

	return util.WriteObject(http.StatusOK, obj, w, req)
}
