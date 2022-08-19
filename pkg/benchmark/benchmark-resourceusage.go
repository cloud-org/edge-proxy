package benchmark

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
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
	Count             int
}

func (r *Resourceusage) Prepare(ctx context.Context) error {
	// start cpu profile
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
	//return r.benchmark_configmap(ctx)
	//return r.benchmark_configmap_count(ctx)
	return r.benchmark_configmap_concurrent(ctx, r.Count) // 25 目前是这个数会高点
}

func (r *Resourceusage) invoke(ctx context.Context) error {
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

	return nil
}

func (r *Resourceusage) benchmark_configmap_count(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(200)
	for i := 0; i < 200; i++ {
		go func() {
			wg.Done()
			if err := r.invoke(ctx); err != nil {
				klog.Errorf("invoke err: %v", err)
			}
		}()
	}

	wg.Wait()

	return nil
}

func (r *Resourceusage) benchmark_configmap_concurrent(ctx context.Context, num int) error {
	go func() {
		klog.Infof("prepare to start cpu profile")
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/profile?seconds=60", "-O", "profile.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("cpu err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()
	defer func() {
		klog.Infof("prepare to start heap profile")
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/heap", "-O", "heap.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("heap err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()

	var count int32
	wg := sync.WaitGroup{}
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func(index int) {
			defer wg.Done()
			timer := time.NewTimer(1 * time.Minute)
			for {
				select {
				case <-timer.C:
					klog.Infof("timer received %v", index)
					return
				default:
					if err := r.invoke(ctx); err != nil {
						klog.Errorf("invoke err: %v", err)
					} else { // 成功
						atomic.AddInt32(&count, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	klog.Infof("count is %v, tps: %v/s", count, count/60)

	return nil
}

func (r *Resourceusage) benchmark_configmap(ctx context.Context) error {
	// start heap pprof for 3m
	timer := time.NewTimer(3 * time.Minute)
	go func() {
		klog.Infof("prepare to start cpu profile")
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/profile?seconds=180", "-O", "profile.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("cpu err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()
	defer func() {
		timer.Stop()
		klog.Infof("prepare to start heap profile")
		_, err := exec.Command("wget", "http://127.0.0.1:10267/debug/pprof/heap", "-O", "heap.txt").CombinedOutput()
		if err != nil {
			klog.Errorf("heap err: %v", err)
			return
		}
		//klog.Infof("data is %v", data)
		return
	}()

	count := 0
	for {
		select {
		case <-timer.C:
			klog.Infof("list resource usage ok, count is %v", count)
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
			count++
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

func NewResourceusage(ns string, proxyClient, client kubernetes.Interface, count int) *Resourceusage {
	return &Resourceusage{
		ProxyClient: proxyClient,
		Client:      client,
		NameSpace:   ns,
		Nums:        1000,
		Count:       count,
		Labels: map[string]string{
			"type":                    "resourceusage",
			util.BENCH_MARK_LABEL_KEY: util.BENCH_MARK_LABEL_VALUE,
		},
		PrepareConfigmaps: make(map[string]*v1.ConfigMap),
	}
}

var _ BenchMarker = &Resourceusage{}
