/*
Copyright 2022 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package benchmark

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark/util"
)

type Filter struct {
	ProxyClient             kubernetes.Interface
	Client                  kubernetes.Interface
	PrepareConfigmaps       map[string]*v1.ConfigMap
	PrepareSkipConfigmaps   map[string]*v1.ConfigMap
	PrepareNoSkipConfigmaps map[string]*v1.ConfigMap
	Labels                  map[string]string
	NameSpace               string
	Nums                    int
}

func NewFilter(ns string, proxyClient, client kubernetes.Interface) *Filter {
	return &Filter{
		ProxyClient:             proxyClient,
		Client:                  client,
		NameSpace:               ns,
		Nums:                    100,
		PrepareConfigmaps:       make(map[string]*v1.ConfigMap),
		PrepareSkipConfigmaps:   make(map[string]*v1.ConfigMap),
		PrepareNoSkipConfigmaps: make(map[string]*v1.ConfigMap),
		Labels: map[string]string{
			"type":                    "filter",
			util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE,
		},
	}
}
func (f *Filter) Prepare(ctx context.Context) error {
	var name string
	for i := 0; i < f.Nums; i++ {
		if i%2 == 1 {
			name = fmt.Sprintf("%s%s-filter-%d", util.SKIPNAME_PREFIX, util.BENCH_MARK_PREFIX, i)
		} else {
			name = fmt.Sprintf("%s-filter-%d", util.BENCH_MARK_PREFIX, i)
		}
		c := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.NameSpace,
				Name:      name,
				Labels:    f.Labels,
			},
			Data: map[string]string{
				"test": fmt.Sprintf("%d", i),
			},
		}
		_, err := f.Client.CoreV1().ConfigMaps(f.NameSpace).Create(ctx, c, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create prepare cm %s error %v", klog.KObj(c), err)
			return err
		}
		key, err := cache.MetaNamespaceKeyFunc(c)
		if err != nil {
			klog.Errorf("get prepare cm %s key error %v", klog.KObj(c), err)
			return err
		}
		if i%2 == 1 {
			f.PrepareSkipConfigmaps[key] = c
		} else {
			f.PrepareNoSkipConfigmaps[key] = c
		}
		f.PrepareConfigmaps[key] = c
	}
	return nil
}

func (f *Filter) benchmark_list_configmaps(ctx context.Context) error {

	cms, err := f.ProxyClient.CoreV1().ConfigMaps(f.NameSpace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(f.Labels).String(),
	})
	if err != nil {
		return err
	}

	if len(cms.Items) != len(f.PrepareNoSkipConfigmaps) {
		klog.Errorf("Get wrong configmap nums, get nums %d prepare num %d", len(cms.Items), len(f.PrepareNoSkipConfigmaps))
		return fmt.Errorf("get wrong configmap nums")
	}

	for _, cm := range cms.Items {
		if strings.HasPrefix(cm.Name, util.SKIPNAME_PREFIX) {
			klog.Errorf("%s has prefix %s", klog.KObj(&cm), util.SKIPNAME_PREFIX)
			return fmt.Errorf("%s has prefix %s", klog.KObj(&cm), util.SKIPNAME_PREFIX)
		}
	}

	return nil
}

func (f *Filter) BenchMark(ctx context.Context) error {
	return f.benchmark_list_configmaps(ctx)
}

func (f *Filter) Clean(ctx context.Context) error {
	for _, cm := range f.PrepareConfigmaps {
		if err := f.Client.CoreV1().ConfigMaps(f.NameSpace).Delete(ctx, cm.GetName(), metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Delete cm %s error %v", klog.KObj(cm), err)
			return err
		}
	}
	return nil
}

func (f *Filter) Name() string {
	return "filter"
}

func (f *Filter) String() string {
	return fmt.Sprintf("benchMark %s", f.Name())
}

var _ BenchMarker = &Filter{}
