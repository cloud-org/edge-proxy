package dev

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	json "github.com/json-iterator/go"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

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

func NewFilterReadCloser(rc io.ReadCloser, resource string, prefix string) (int, io.ReadCloser, error) {
	sfrc := &skipListFilterReadCloser{
		data: new(bytes.Buffer),
		rc:   rc,
	}

	switch resource {
	case "pods":
		var podList v1.PodList
		err := json.NewDecoder(rc).Decode(&podList)
		if err != nil {
			klog.Errorf("%s decode err: %v", resource, err)
			return 0, nil, err
		}
		var items []v1.Pod
		for i := 0; i < len(podList.Items); i++ {
			if !strings.HasPrefix(podList.Items[i].Name, prefix) {
				items = append(items, podList.Items[i])
			}
		}
		podList.Items = items
		marshalBytes, err := json.Marshal(podList)
		if err != nil {
			klog.Errorf("%s marshal err: %v", resource, err)
			return 0, nil, err
		}
		sfrc.data = bytes.NewBuffer(marshalBytes)
		return len(marshalBytes), sfrc, nil
	case "configmaps":
		var configmaps v1.ConfigMapList
		err := json.NewDecoder(rc).Decode(&configmaps)
		if err != nil {
			klog.Errorf("%s decode err: %v", resource, err)
			return 0, nil, err
		}
		var items []v1.ConfigMap
		for i := 0; i < len(configmaps.Items); i++ {
			if !strings.HasPrefix(configmaps.Items[i].Name, prefix) {
				items = append(items, configmaps.Items[i])
			}
		}
		configmaps.Items = items
		marshalBytes, err := json.Marshal(configmaps)
		if err != nil {
			klog.Errorf("%s marshal err: %v", resource, err)
			return 0, nil, err
		}
		sfrc.data = bytes.NewBuffer(marshalBytes)
		return len(marshalBytes), sfrc, nil
	default:
		return 0, nil, fmt.Errorf("err resource type")
	}

}
