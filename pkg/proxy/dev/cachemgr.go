package dev

import (
	"fmt"
	"io"
	"path/filepath"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util/storage"

	json "github.com/json-iterator/go"
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
}

// NewCacheMgr create a cachemgr
func NewCacheMgr(s storage.Store) *CacheMgr {
	return &CacheMgr{
		storage: s,
		memdata: make(map[string][]byte),
	}
}

//CacheResponseMem handle pod and configmaps mem cache
// info: req inject requestInfo
// prc: is a readCloser
// labelType: filter resource label type or generate unique key for cache
func (c *CacheMgr) CacheResponseMem(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {
	key := KeyFunc(info.Resource, info.Namespace, labelType)

	//p := new(bytes.Buffer)
	//p := bytes.NewBuffer(make([]byte, 0, 100*1024)) // data.len: 1123875 线上测评数据
	//_, err := p.ReadFrom(prc)
	data, err := io.ReadAll(prc)
	//data, err := ReadAll(prc)
	if err != nil {
		klog.Errorf("read prc err: %v", err)
		return err
	}

	//data := p.Bytes()
	c.memdata[key] = data

	klog.Infof("%s memdata create ok, data.len: %v, data.cap: %v", info.Resource, len(data), cap(data))

	return nil
}

//CacheResponse cache consistency list data
func (c *CacheMgr) CacheResponse(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {
	switch info.Resource {
	case "pods":
		var podList v1.PodList
		err := json.NewDecoder(prc).Decode(&podList)
		if err != nil {
			klog.Errorf("%s decode err: %v", info.Resource, err)
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
		marshalBytes, err := json.Marshal(podList)
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
		var configmaps v1.ConfigMapList
		err := json.NewDecoder(prc).Decode(&configmaps)
		if err != nil {
			klog.Errorf("%s decode err: %v", info.Resource, err)
			return err
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configmaps.Items); i++ {
			// filter label type
			if configmaps.Items[i].Labels["type"] == labelType {
				//klog.Infof("add item %s", configmaps.Items[i].Name)
				items = append(items, configmaps.Items[i])
			}
		}
		configmaps.Items = items
		marshalBytes, err := json.Marshal(configmaps)
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
