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
	"time"

	json "github.com/json-iterator/go"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientgov1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark/util"
)

type Functional struct {
	ProxyClient          kubernetes.Interface
	Client               kubernetes.Interface
	PrepareConfigmaps    map[string]*v1.ConfigMap
	Labels               map[string]string
	ProxyConfigMapLister clientgov1.ConfigMapLister
	ConfigMapLister      clientgov1.ConfigMapLister
	NameSpace            string
	Nums                 int
}

func NewFunctional(ns string, proxyClient, client kubernetes.Interface, proxylister, lister clientgov1.ConfigMapLister) *Functional {
	return &Functional{
		ProxyClient:          proxyClient,
		Client:               client,
		NameSpace:            ns,
		Nums:                 100,
		ProxyConfigMapLister: proxylister,
		ConfigMapLister:      lister,
		PrepareConfigmaps:    make(map[string]*v1.ConfigMap),
		Labels: map[string]string{
			"type":                    "functional",
			util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE,
		},
	}
}

func (f *Functional) Prepare(ctx context.Context) error {
	var name string
	for i := 0; i < f.Nums; i++ {
		name = fmt.Sprintf("%s-functional-%d", util.BENCH_MARK_PREFIX, i)
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
		f.PrepareConfigmaps[key] = c
	}
	return nil

}

func (f *Functional) benchmark_list_configmaps(ctx context.Context) error {

	cms, err := f.ProxyClient.CoreV1().ConfigMaps(f.NameSpace).List(
		ctx, metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(f.Labels).String(),
		})
	if err != nil {
		return err
	}

	if len(f.PrepareConfigmaps) != len(cms.Items) {
		klog.Errorf("Inconsistent data returned, prepare len %d get len %d", len(f.PrepareConfigmaps), len(cms.Items))
		return fmt.Errorf("inconsistent data returned")
	}
	for i, c := range cms.Items {
		key, err := cache.MetaNamespaceKeyFunc(&cms.Items[i])
		if err != nil {
			klog.Errorf("get cm %s key error %v", klog.KObj(&c), err)
			return err
		}
		if _, ok := f.PrepareConfigmaps[key]; !ok {
			klog.Errorf("Can not find %s from prepare configmaps", key)
			return fmt.Errorf("can not find %s from prepare configmaps", key)
		}
	}
	return nil
}

func (f *Functional) benchmark_watch_configmaps(ctx context.Context) error {

	labelKey := "watch"
	labelFinalValue := "success"
	getCMPatchData := func(str string) map[string]interface{} {
		if len(str) == 0 {
			str = uuid.New().String()
		}
		return map[string]interface{}{
			"metadata": map[string]map[string]string{
				"labels": {
					labelKey: str,
				}}}
	}

	for _, p := range f.PrepareConfigmaps {
		for i := 0; i < 3; i++ {
			playLoadBytes, err := json.Marshal(getCMPatchData(""))
			if err != nil {
				klog.Errorf("Marshal configmap patch data error %v", err)
				continue
			}
			if _, err := f.Client.CoreV1().ConfigMaps(p.Namespace).Patch(ctx, p.Name,
				types.StrategicMergePatchType, playLoadBytes,
				metav1.PatchOptions{}); err != nil {
				klog.Errorf("Patch configmap %s error %v", klog.KObj(p), err)
				// do not return
				return err
			}
			time.Sleep(time.Millisecond * 50)
		}
	}

	for _, p := range f.PrepareConfigmaps {
		playLoadBytes, err := json.Marshal(getCMPatchData(labelFinalValue))
		if err != nil {
			klog.Errorf("Marshal configmap patch data error %v", err)
			return err
		}
		if _, err := f.Client.CoreV1().ConfigMaps(p.Namespace).Patch(ctx, p.Name,
			types.StrategicMergePatchType, playLoadBytes,
			metav1.PatchOptions{}); err != nil {
			klog.Errorf("Patch configmap %s error %v", klog.KObj(p), err)
			return nil
		}
		time.Sleep(time.Millisecond * 50)
	}

	time.Sleep(time.Second * 5)

	for _, p := range f.PrepareConfigmaps {

		fp, err := f.ProxyConfigMapLister.ConfigMaps(p.Namespace).Get(p.Name)
		if err != nil {
			klog.Errorf("ProxyPodLister get cms error %v", err)
			return err
		}

		klog.V(4).Infof("cm %s labels %++v", klog.KObj(fp), fp.Labels)
		if v, ok := fp.Labels[labelKey]; !ok {
			return fmt.Errorf("Can not find label key %s", labelKey)
		} else if v != labelFinalValue {
			return fmt.Errorf("Can not label key %s value is %s not %s", labelKey, v, labelFinalValue)
		}
	}

	return nil
}

func (f *Functional) BenchMark(ctx context.Context) error {
	if err := f.benchmark_list_configmaps(ctx); err != nil {
		klog.Errorf("list configmap error %v", err)
		return err
	}
	if err := f.benchmark_watch_configmaps(ctx); err != nil {
		klog.Errorf("watch config map benchmark error %v", err)
		return err
	}
	return nil
}

func (f *Functional) Clean(ctx context.Context) error {
	for _, cm := range f.PrepareConfigmaps {
		if err := f.Client.CoreV1().ConfigMaps(f.NameSpace).Delete(ctx, cm.GetName(), metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Delete cm %s error %v", klog.KObj(cm), err)
			return err
		}
	}
	return nil
}

func (f *Functional) Name() string {
	return "functional"
}

func (f *Functional) String() string {
	return fmt.Sprintf("benchMark %s", f.Name())
}

var _ BenchMarker = &Functional{}
