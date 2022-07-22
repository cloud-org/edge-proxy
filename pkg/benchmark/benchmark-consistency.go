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
	"os/exec"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark/util"
)

type Consistency struct {
	ProxyClient       kubernetes.Interface
	Client            kubernetes.Interface
	PrepareConfigmaps map[string]*v1.ConfigMap
	NameSpace         string
	Labels            map[string]string
	Nums              int
}

func NewConsistency(ns string, proxyClient, client kubernetes.Interface) *Consistency {
	return &Consistency{
		ProxyClient: proxyClient,
		Client:      client,
		NameSpace:   ns,
		Nums:        100,
		Labels: map[string]string{
			"type":                    "consistency",
			util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE,
		},
		PrepareConfigmaps: make(map[string]*v1.ConfigMap),
	}
}

func (c *Consistency) Prepare(ctx context.Context) error {

	for i := 0; i < c.Nums; i++ {

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: c.NameSpace,
				Name:      fmt.Sprintf("%s-consistency-%d", util.BENCH_MARK_PREFIX, i),
				Labels:    c.Labels,
			},
			Data: map[string]string{
				"test": fmt.Sprintf("%d", i),
			},
		}
		_, err := c.ProxyClient.CoreV1().ConfigMaps(c.NameSpace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create prepare cm %s error %v", klog.KObj(cm), err)
			return err
		}
		key, err := cache.MetaNamespaceKeyFunc(cm)
		if err != nil {
			klog.Errorf("get prepare cm %s key error %v", klog.KObj(cm), err)
			return err
		}
		c.PrepareConfigmaps[key] = cm
	}

	c.ProxyClient.CoreV1().ConfigMaps(c.NameSpace).List(
		ctx, metav1.ListOptions{})

	c.ProxyClient.CoreV1().ConfigMaps(c.NameSpace).List(
		ctx, metav1.ListOptions{})

	// iptables
	cmd := "iptables -I OUTPUT -p tcp --dport 6443 -j DROP"
	data, err := exec.Command("iptables",
		"-I", "OUTPUT", "-p", "tcp", "--dport", "6443", "-j", "DROP").CombinedOutput()
	if err != nil {
		klog.Errorf("exec %s error %v", cmd, err)
		// manually exec iptables drop
		//return err
	}
	klog.Infof("exec %s output %s", cmd, string(data))

	// must sleep 30s, edge-hub need find Network disconnection
	time.Sleep(30 * time.Second)
	return nil
}

func (c *Consistency) BenchMark(ctx context.Context) error {
	return c.benchmark_configmap(ctx)
}

func (r *Consistency) benchmark_configmap(ctx context.Context) error {
	cms, err := r.ProxyClient.CoreV1().ConfigMaps(r.NameSpace).List(
		ctx, metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(r.Labels).String(),
		})
	if err != nil {
		klog.Errorf("%s proxy list cms error %v", r, err)
		return err
	}

	proxyCMs := make(map[string]*v1.ConfigMap)

	for i, c := range cms.Items {
		key, err := cache.MetaNamespaceKeyFunc(&cms.Items[i])
		if err != nil {
			klog.Errorf("%s get cm %s key error %v", r, klog.KObj(&c), err)
			return err
		}
		proxyCMs[key] = &cms.Items[i]
		if _, ok := r.PrepareConfigmaps[key]; !ok {
			klog.Errorf("can not find %s from prepare configmaps", key)
			return fmt.Errorf("%s can not find %s from prepare configmaps", r, key)
		}
	}

	for key, _ := range r.PrepareConfigmaps {
		if _, ok := proxyCMs[key]; !ok {
			klog.Errorf("can not find %s from proxy configmaps", key)
			return fmt.Errorf("%s can not find %s from proxy configmaps", r, key)
		}
	}
	return nil
}

func (c *Consistency) Clean(ctx context.Context) error {
	for _, cm := range c.PrepareConfigmaps {
		if err := c.Client.CoreV1().ConfigMaps(c.NameSpace).Delete(ctx, cm.GetName(), metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Delete cm %s error %v", klog.KObj(cm), err)
			return err
		}
	}
	return nil
}

func (c *Consistency) Name() string {
	return "consistency"
}

func (c *Consistency) String() string {
	return fmt.Sprintf("benchMark %s", c.Name())
}

var _ BenchMarker = &Consistency{}
