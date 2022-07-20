package dev

import (
	"bytes"
	"io"
	"strings"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type skipListFilter struct {
	resource   string // pod or configmap
	prefix     string // 目前默认为: skip-
	serializer *serializer.Serializer
}

func NewSkipListFilter(resource string, serializer *serializer.Serializer) *skipListFilter {
	return &skipListFilter{
		resource:   resource,
		prefix:     "skip-",
		serializer: serializer,
	}
}

// ObjectResponseFilter filter the endpoints from get response object and return the bytes
func (sf *skipListFilter) ObjectResponseFilter(b []byte) ([]byte, error) {
	eps, err := sf.serializer.Decode(b)
	if err != nil || eps == nil {
		klog.Errorf("skip filter, failed to decode response in ObjectResponseFilter of endpointsFilterHandler, %v", err)
		return b, nil
	}

	switch sf.resource {
	case "pods":
		podList, ok := eps.(*v1.PodList)
		if !ok {
			klog.Errorf("*v1.PodList 断言失败")
			return b, nil
		}
		var items []v1.Pod
		for i := 0; i < len(podList.Items); i++ {
			if !strings.HasPrefix(podList.Items[i].Name, sf.prefix) {
				items = append(items, podList.Items[i])
			}
		}
		podList.Items = items
		return sf.serializer.Encode(podList)
	case "configmaps":
		configMapList, ok := eps.(*v1.ConfigMapList)
		if !ok {
			klog.Errorf("*v1.ConfigMapList 断言失败")
			return b, nil
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configMapList.Items); i++ {
			if !strings.HasPrefix(configMapList.Items[i].Name, sf.prefix) {
				items = append(items, configMapList.Items[i])
			}
		}
		configMapList.Items = items
		return sf.serializer.Encode(configMapList)
	default:
		klog.Infof("未知的 resource: %v, 不进行过滤", sf.resource)
		return b, nil
	}
}

// todo: 暂时不实现
// FilterWatchObject filter the endpoints from watch response object and return the bytes
func (sf *skipListFilter) StreamResponseFilter(rc io.ReadCloser, ch chan watch.Event) error {
	return nil
}

type skipListFilterReadCloser struct {
	data *bytes.Buffer
	rc   io.ReadCloser
}

// todo: 暂时不考虑 watch
func (s *skipListFilterReadCloser) Read(p []byte) (int, error) {
	return s.data.Read(p)
}

// Close close readers
func (s *skipListFilterReadCloser) Close() error {
	return s.rc.Close()
}

func NewFilterReadCloser(rc io.ReadCloser, sf *skipListFilter) (int, io.ReadCloser, error) {
	sfrc := &skipListFilterReadCloser{
		data: new(bytes.Buffer),
		rc:   rc,
	}

	var newData []byte
	n, err := sfrc.data.ReadFrom(rc)
	if err != nil {
		return int(n), sfrc, err
	}

	newData, err = sf.ObjectResponseFilter(sfrc.data.Bytes())
	sfrc.data = bytes.NewBuffer(newData)

	return len(newData), sfrc, err
}
