package dev

import (
	"fmt"
	"io"
	"path/filepath"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/types"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util/storage"

	json "github.com/json-iterator/go"
	v1 "k8s.io/api/core/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

type CacheMgr struct {
	storage storage.Store     // disk cache manager for consistency list
	memdata map[string][]byte // mem cache for resourceusage list
}

func NewCacheMgr(s storage.Store) *CacheMgr {
	return &CacheMgr{
		storage: s,
		memdata: make(map[string][]byte),
	}
}

// ReadAll tag: v0.0.26 有打包使用 score: 130241.5284
func ReadAll(r io.Reader) ([]byte, error) {
	//b := make([]byte, 0, 532874) // local minikube data.len
	b := make([]byte, 0, 1133672) // data.len: 1133657 线上测评数据

	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

func (c *CacheMgr) CacheResponseMemNew(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {

	var data []byte
	var err error
	switch info.Resource {
	case "pods":
		data, err = io.ReadAll(prc)
		if err != nil {
			klog.Errorf("read all err: %v", err)
			return err
		}
	case "configmaps":
		var configmaps types.ConfigMapList
		err = json.NewDecoder(prc).Decode(&configmaps)
		if err != nil {
			klog.Errorf("%s decode err: %v", info.Resource, err)
			return err
		}

		data, err = json.Marshal(configmaps)
		if err != nil {
			klog.Errorf("%s marshal err: %v", info.Resource, err)
			return err
		}
	}
	key := KeyFunc(info.Resource, info.Namespace, labelType)
	//if err = c.storage.Create(key, marshalBytes); err != nil {
	// klog.Errorf("storage create err: %v", err)
	// return err
	//}
	//klog.Infof("%s storage create ok", info.Resource)

	//data := p.Bytes()
	c.memdata[key] = data

	klog.Infof("%s memdata create ok, data.len: %v, data.cap: %v", info.Resource, len(data), cap(data))

	return nil
}

//CacheResponseMem cache resourceusage list data
func (c *CacheMgr) CacheResponseMem(info *apirequest.RequestInfo, prc io.ReadCloser, labelType string) error {
	key := KeyFunc(info.Resource, info.Namespace, labelType)

	//p := new(bytes.Buffer)
	//p := bytes.NewBuffer(make([]byte, 0, 100*1024)) // data.len: 1123875 线上测评数据
	//_, err := p.ReadFrom(prc)
	//data, err := io.ReadAll(prc)
	data, err := ReadAll(prc)
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

// KeyFunc combine comp resource ns name into a key
func KeyFunc(resource, ns, labelType string) string {
	comp := "bench"
	return filepath.Join(comp, resource, ns, labelType)
}
