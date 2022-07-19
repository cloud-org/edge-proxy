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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"

	"code.aliyun.com/openyurt/edge-proxy/cmd/benchmark/app/options"
	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark/util"
)

type BenchMark struct {
	// The Client directly interacts with APIServer
	Client kubernetes.Interface
	// The ProxyClient interacts with the Edge-proxy
	ProxyClient          kubernetes.Interface
	ProxyConfigMapLister v1.ConfigMapLister
	ConfigMapLister      v1.ConfigMapLister
	SubBenchMarkers      []BenchMarker
	Namespace            string
}

func NewBenchMark(deps *options.BenchMarkOptions) (*BenchMark, error) {
	proxyConfigFile := "/edge-proxy.kubeconfig"
	if err := util.CreateProxyKubeConfigFile(proxyConfigFile); err != nil {
		klog.Errorf("Create edge-proxy kubeconfigfile %s error %v", proxyConfigFile, err)
		return nil, err
	}

	proxyc, err := clientcmd.BuildConfigFromFlags("", proxyConfigFile)
	if err != nil {
		return nil, fmt.Errorf("build prox config from file error %v", err)
	}

	proxyc.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 20000)
	// edge-proxy client
	proxycs, err := kubernetes.NewForConfig(proxyc)
	if err != nil {
		klog.Errorf("NewForConfig for proxy error %v", err)
		return nil, err
	}

	// in cluster config
	// todo: 从 kubeconfig path 读取
	c, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, fmt.Errorf("build config from flags error %v", err)
	}

	// set rate limit
	c.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 20000)
	// 实例化clientset对象
	cs, err := kubernetes.NewForConfig(c)
	if err != nil {
		klog.Errorf("NewForConfig error %v", err)
		return nil, err
	}

	proxyf := informers.NewSharedInformerFactory(proxycs, 0)
	proxyInformerCMs := proxyf.Core().V1().ConfigMaps()
	proxyCMInformer := proxyInformerCMs.Informer()

	klog.Infof("Proxy informer factory prepare to start ...")
	proxyf.Start(wait.NeverStop)
	klog.Infof("Proxy informer started")

	if !cache.WaitForCacheSync(wait.NeverStop, proxyCMInformer.HasSynced) {
		klog.Exitf("Proxy informer timed out waiting for caches to sync")
	}
	klog.Infof("Proxy informer wait for cache synced")

	f := informers.NewSharedInformerFactory(cs, 0)
	informerCMs := f.Core().V1().ConfigMaps()
	cmInformer := informerCMs.Informer()

	klog.Infof("Informer factory prepare to start ...")
	f.Start(wait.NeverStop)
	klog.Infof("Informer started")

	if !cache.WaitForCacheSync(wait.NeverStop, cmInformer.HasSynced) {
		klog.Exitf("Informer timed out waiting for caches to sync")
	}
	klog.Infof("Informer wait for cache synced")

	b := &BenchMark{
		Client:               cs,
		ProxyClient:          proxycs,
		ProxyConfigMapLister: proxyInformerCMs.Lister(),
		ConfigMapLister:      informerCMs.Lister(),
		SubBenchMarkers:      make([]BenchMarker, 0, 4),
		Namespace:            deps.Namespace,
	}

	b.SubBenchMarkers = append(b.SubBenchMarkers,
		NewFunctional(b.Namespace, proxycs, cs, b.ProxyConfigMapLister, b.ConfigMapLister),
		NewFilter(b.Namespace, proxycs, cs),
		NewConsistency(b.Namespace, proxycs, cs))
	return b, nil
}

func (m *BenchMark) Run(ctx context.Context) error {
	if err := m.Prepare(ctx); err != nil {
		return err
	}

	// sleep until edge-proxy ready

	for _, b := range m.SubBenchMarkers {
		klog.Infof("######## Start to %s ########", b)

		if err := b.Prepare(ctx); err != nil {
			klog.Errorf("%s prepare error %v", b, err)
			return err
		}
		if err := b.BenchMark(ctx); err != nil {
			klog.Errorf("%s error %v", b, err)
			b.Clean(ctx)
			continue
		}

		if err := b.Clean(ctx); err != nil {
			klog.Errorf("%s clean error %v", b, err)
			return err
		}
		klog.Infof("%s successfully ...", b)
	}

	klog.Infof("-------- All benchmark exec end --------")
	return nil
}

func (m *BenchMark) Prepare(ctx context.Context) error {
	labelResource := labels.SelectorFromSet(map[string]string{
		util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE})

	listOptions := metav1.ListOptions{
		LabelSelector: labelResource.String(),
	}
	if err := m.Client.CoreV1().ConfigMaps(m.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Clean configmaps error: %v", err)
			return err
		}
	}
	if err := m.Client.CoreV1().Pods(m.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Clean pods error: %v", err)
			return err
		}
	}
	return nil
}
