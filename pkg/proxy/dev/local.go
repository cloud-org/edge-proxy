package dev

import (
	"fmt"
	"net/http"
	"sync"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// IsHealthy is func for fetching healthy status of remote server
type IsHealthy func() bool

// LocalProxy is responsible for handling requests when remote servers are unhealthy
type LocalProxy struct {
	sync.RWMutex
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
		default: // list, get, update
			err = lp.localReqCache(w, req)
		}

		if err != nil {
			klog.Errorf("could not proxy local for %s %v", reqInfo.Resource, err)
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		klog.Errorf("request(%s) is not supported when cluster is unhealthy", reqInfo.Resource)
		w.WriteHeader(http.StatusForbidden)
	}
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

	if lp.cacheMgr == nil {
		klog.Errorf("cache mgr is nil")
		return fmt.Errorf("get cache mgr err")
	}

	obj, err := lp.cacheMgr.QueryCache(info, consistencyType)
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

// IsHealthy always return true
func (lp *LocalProxy) IsHealthy() bool {
	return true
}
