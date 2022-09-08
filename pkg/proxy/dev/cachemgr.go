package dev

import (
	"fmt"
	"io"
	"path/filepath"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util/storage"

	v1 "k8s.io/api/core/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

//CacheMgr cache for list resp.Body
type CacheMgr struct {
	//storage disk cache manager for consistency list
	storage storage.Store
	//memdata memory cache for list labelSelector result
	memdata map[string][]byte
	// serializerManager for apiserver resp.Body encode and decode
	serializerManager *serializer.SerializerManager
}

// NewCacheMgr create a cachemgr
func NewCacheMgr(s storage.Store, serializerManager *serializer.SerializerManager) *CacheMgr {
	return &CacheMgr{
		storage:           s,
		memdata:           make(map[string][]byte),
		serializerManager: serializerManager,
	}
}

//CacheResponseMem handle pod and configmaps mem cache
// info: req inject requestInfo
// prc: is a readCloser
// labelType: filter resource label type or generate unique key for cache
func (c *CacheMgr) CacheResponseMem(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {
	key := KeyFunc(info.Resource, info.Namespace, labelType)

	data, err := io.ReadAll(prc)
	//data, err := ReadAll(prc)
	if err != nil {
		klog.Errorf("read prc err: %v", err)
		return err
	}

	c.memdata[key] = data

	klog.Infof("%s memdata create ok, data.len: %v, data.cap: %v", info.Resource, len(data), cap(data))

	return nil
}

//CacheResponse cache consistency list data
func (c *CacheMgr) CacheResponse(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {
	switch info.Resource {
	case "pods":
		data, err := io.ReadAll(prc)
		if err != nil {
			klog.Errorf("readAll err: %v", err)
			return err
		}
		serializerObject := CreateSerializer(info, c.serializerManager)
		obj, err := serializerObject.Decode(data)
		if err != nil {
			klog.Errorf("%s decode err: %v", info.Resource, err)
			return err
		}
		podList, ok := obj.(*v1.PodList)
		if !ok {
			klog.Errorf("*v1.PodList 断言失败")
			return err
		}
		var items []v1.Pod
		for i := 0; i < len(podList.Items); i++ {
			// filter label type
			if podList.Items[i].Labels["type"] == labelType {
				//klog.Infof("add item %s", podList.Items[i].Name)
				items = append(items, podList.Items[i])
			}
		}
		podList.Items = items
		marshalBytes, err := serializerObject.Encode(podList)
		if err != nil {
			klog.Errorf("%s marshal err: %v", info.Resource, err)
			return err
		}
		key := KeyFunc(info.Resource, info.Namespace, labelType)
		if err = c.storage.Create(key, marshalBytes); err != nil {
			klog.Errorf("%s storage create err: %v", info.Resource, err)
			return err
		}
		klog.Infof("%s storage create ok", info.Resource)
	case "configmaps":
		data, err := io.ReadAll(prc)
		if err != nil {
			klog.Errorf("readAll err: %v", err)
			return err
		}
		serializerObject := CreateSerializer(info, c.serializerManager)
		obj, err := serializerObject.Decode(data)
		if err != nil {
			klog.Errorf("%s decode err: %v", info.Resource, err)
			return err
		}
		configmapList, ok := obj.(*v1.ConfigMapList)
		if !ok {
			klog.Errorf("*v1.PodList 断言失败")
			return err
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configmapList.Items); i++ {
			// filter label type
			if configmapList.Items[i].Labels["type"] == labelType {
				items = append(items, configmapList.Items[i])
			}
		}
		configmapList.Items = items
		marshalBytes, err := serializerObject.Encode(configmapList)
		if err != nil {
			klog.Errorf("%s marshal err: %v", info.Resource, err)
			return err
		}
		key := KeyFunc(info.Resource, info.Namespace, labelType)
		if err = c.storage.Create(key, marshalBytes); err != nil {
			klog.Errorf("storage create err: %v", err)
			return err
		}
		klog.Infof("%s storage create ok", info.Resource)
	default:
		return fmt.Errorf("err resource type: %s", info.Resource)
	}

	return nil
}

//QueryCache query for consistency list data
func (c *CacheMgr) QueryCache(info *apirequest.RequestInfo, labelType string) ([]byte, error) {
	key := KeyFunc(info.Resource, info.Namespace, labelType)
	return c.storage.Get(key)
}

//QueryCacheMem query for resourceusage list data
func (c *CacheMgr) QueryCacheMem(resource, ns, labelType string) ([]byte, bool) {
	key := KeyFunc(resource, ns, labelType)
	data, ok := c.memdata[key]
	return data, ok
}

// KeyFunc generate a key for cache manager
func KeyFunc(resource, ns, labelType string) string {
	comp := "bench"
	return filepath.Join(comp, resource, ns, labelType)
}
