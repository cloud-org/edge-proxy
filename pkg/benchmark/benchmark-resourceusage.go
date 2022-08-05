package benchmark

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"code.aliyun.com/openyurt/edge-proxy/pkg/benchmark/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Resourceusage struct {
	ProxyClient       kubernetes.Interface
	Client            kubernetes.Interface
	PrepareConfigmaps map[string]*v1.ConfigMap
	NameSpace         string
	Labels            map[string]string
	Nums              int
}

func (r *Resourceusage) Prepare(ctx context.Context) error {
	// start cpu profile
	klog.Infof("prepare to start cpu profile")
	go func() {
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/profile?seconds=60", "-O", "profile.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("cpu err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()
	for i := 0; i < r.Nums; i++ {

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: r.NameSpace,
				Name:      fmt.Sprintf("%s-resource-%d", util.BENCH_MARK_PREFIX, i),
				Labels:    r.Labels,
			},
			Data: map[string]string{
				"test": fmt.Sprintf("%d", i),
			},
		}
		_, err := r.ProxyClient.CoreV1().ConfigMaps(r.NameSpace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create prepare cm %s error %v", klog.KObj(cm), err)
			return err
		}
		key, err := cache.MetaNamespaceKeyFunc(cm)
		if err != nil {
			klog.Errorf("get prepare cm %s key error %v", klog.KObj(cm), err)
			return err
		}
		r.PrepareConfigmaps[key] = cm
	}

	return nil
}

func (r *Resourceusage) BenchMark(ctx context.Context) error {
	return r.benchmark_configmap(ctx)
}

func (r *Resourceusage) benchmark_configmap(ctx context.Context) error {
	// start heap pprof for 1m
	timer := time.NewTimer(1 * time.Minute)
	defer func() {
		timer.Stop()
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/heap", "-O", "heap.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("heap err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()
	klog.Infof("prepare to start heap profile")

	for {
		select {
		case <-timer.C:
			klog.Infof("list resource usage ok")
			return nil
		default:
			// list cms
			cms, err := r.ProxyClient.CoreV1().ConfigMaps(r.NameSpace).List(
				ctx, metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(r.Labels).String(),
				})
			if err != nil {
				return err
			}
			if len(r.PrepareConfigmaps) != len(cms.Items) {
				klog.Errorf(
					"Inconsistent data returned, prepare len %d get len %d",
					len(r.PrepareConfigmaps),
					len(cms.Items),
				)
				return fmt.Errorf("inconsistent data returned")
			}
		}
	}
}

func (r *Resourceusage) Clean(ctx context.Context) error {
	for _, cm := range r.PrepareConfigmaps {
		if err := r.Client.CoreV1().ConfigMaps(r.NameSpace).Delete(ctx, cm.GetName(), metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Delete cm %s error %v", klog.KObj(cm), err)
			return err
		}
	}
	return nil
}

func (r *Resourceusage) Name() string {
	return "resourceusage"
}

func (r *Resourceusage) String() string {
	return fmt.Sprintf("benchMark %s", r.Name())
}

func NewResourceusage(ns string, proxyClient, client kubernetes.Interface) *Resourceusage {
	return &Resourceusage{
		ProxyClient: proxyClient,
		Client:      client,
		NameSpace:   ns,
		Nums:        1000,
		Labels: map[string]string{
			"type":                    "resourceusage",
			util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE,
		},
		PrepareConfigmaps: make(map[string]*v1.ConfigMap),
	}
}

var _ BenchMarker = &Resourceusage{}
