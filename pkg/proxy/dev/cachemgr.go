package dev

import (
	"fmt"
	"io"
	"path/filepath"

	json "github.com/json-iterator/go"
	"github.com/openyurtio/openyurt/pkg/yurthub/storage"
	v1 "k8s.io/api/core/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

type CacheMgr struct {
	storage storage.Store
}

func NewCacheMgr(s storage.Store) *CacheMgr {
	return &CacheMgr{
		storage: s,
	}
}

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
		// todo: 暂时使用 create 进行测试
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
		// todo: 暂时使用 create 进行测试
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

func (c *CacheMgr) QueryCache(info *apirequest.RequestInfo, labelType string) ([]byte, error) {
	key := KeyFunc(info.Resource, info.Namespace, labelType)
	return c.storage.Get(key)
}

// KeyFunc combine comp resource ns name into a key
func KeyFunc(resource, ns, labelType string) string {
	comp := "bench"
	return filepath.Join(comp, resource, ns, labelType)
}
