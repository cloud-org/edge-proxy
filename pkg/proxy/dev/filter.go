package dev

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

//skipListFilterReadCloser for filter benchmark
// data: real return data
// rc: list filter apiserver resp io.ReadCloser
type skipListFilterReadCloser struct {
	data *bytes.Buffer
	rc   io.ReadCloser
}

// Read read data from s.data to p
func (s *skipListFilterReadCloser) Read(p []byte) (int, error) {
	return s.data.Read(p)
}

// Close close resp reader
func (s *skipListFilterReadCloser) Close() error {
	return s.rc.Close()
}

// NewFilterReadCloser filter prefix for rc
// rc: list filter apiserver resp io.ReadCloser(resp.Body)
// resource: maybe configmaps/pods
// prefix: it should be "skip-" in order to pass filter benchmark
func NewFilterReadCloser(rc io.ReadCloser, resource string, prefix string, serializerObject *serializer.Serializer) (int, io.ReadCloser, error) {
	sfrc := &skipListFilterReadCloser{
		data: new(bytes.Buffer),
		rc:   rc,
	}

	switch resource {
	case "pods":
		data, err := io.ReadAll(rc)
		if err != nil {
			klog.Errorf("readAll err: %v", err)
			return 0, nil, err
		}
		obj, err := serializerObject.Decode(data)
		if err != nil {
			klog.Errorf("%s decode err: %v", resource, err)
			return 0, nil, err
		}
		podList, ok := obj.(*v1.PodList)
		if !ok {
			klog.Errorf("*v1.PodList 断言失败")
			return 0, nil, nil
		}
		var items []v1.Pod
		for i := 0; i < len(podList.Items); i++ {
			// if name doesn't include prefix, then append to the items
			if !strings.HasPrefix(podList.Items[i].Name, prefix) {
				items = append(items, podList.Items[i])
			}
		}
		podList.Items = items
		marshalBytes, err := serializerObject.Encode(podList)
		if err != nil {
			klog.Errorf("encode err: %v", err)
			return 0, nil, err
		}
		sfrc.data = bytes.NewBuffer(marshalBytes)
		return len(marshalBytes), sfrc, nil
	case "configmaps":
		data, err := io.ReadAll(rc)
		if err != nil {
			klog.Errorf("readAll err: %v", err)
			return 0, nil, err
		}
		obj, err := serializerObject.Decode(data)
		if err != nil {
			klog.Errorf("%s decode err: %v", resource, err)
			return 0, nil, err
		}
		configmapList, ok := obj.(*v1.ConfigMapList)
		if !ok {
			klog.Errorf("*v1.ConfigMapList 断言失败")
			return 0, nil, nil
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configmapList.Items); i++ {
			// if name doesn't include prefix, then append to the items
			if !strings.HasPrefix(configmapList.Items[i].Name, prefix) {
				items = append(items, configmapList.Items[i])
			}
		}
		configmapList.Items = items
		marshalBytes, err := serializerObject.Encode(configmapList)
		if err != nil {
			klog.Errorf("encode err: %v", err)
			return 0, nil, err
		}
		sfrc.data = bytes.NewBuffer(marshalBytes)
		return len(marshalBytes), sfrc, nil
	default:
		return 0, nil, fmt.Errorf("err resource type: %s", resource)
	}

}
