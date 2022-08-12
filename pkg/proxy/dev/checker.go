package dev

import (
	"net/url"
	"sync"
	"time"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/health"
	"k8s.io/klog/v2"
)

// checker for check remote server health or not
type checker struct {
	sync.RWMutex
	remoteServer   *url.URL
	clusterHealthy bool
	lastTime       time.Time
}

func NewChecker(remoteServer *url.URL) *checker {
	return &checker{
		remoteServer: remoteServer,
		lastTime:     time.Now(),
	}
}

func (c *checker) start(stopCh <-chan struct{}) {
	c.check() // check once when start
	go c.loop(stopCh)
}

func (c *checker) loop(stopCh <-chan struct{}) {
	checkDuration := 10 * time.Second
	timer := time.NewTimer(checkDuration)
	defer timer.Stop()
	for {
		timer.Reset(checkDuration)
		select {
		case <-stopCh:
			klog.Infof("check exit when received stopCh close")
			return
		case <-timer.C:
			//klog.Infof("checker timer received")
			c.check()
		}
	}
}

func (c *checker) check() {
	if !health.CheckClusterIsHealthyByGet(c.remoteServer.String()) {
		c.markAsUnhealthy()
		return
	}

	c.markAsHealthy()
}

func (c *checker) markAsHealthy() {
	c.setHealthy(true)
	now := time.Now()
	c.lastTime = now
}

func (c *checker) markAsUnhealthy() {
	if c.isHealthy() {
		c.setHealthy(false)
		now := time.Now()
		klog.Infof(
			"cluster becomes unhealthy from %v, healthy status lasts %v, remote server: %v",
			time.Now(),
			now.Sub(c.lastTime),
			c.remoteServer.String(),
		)
		c.lastTime = now
	}
}

func (c *checker) isHealthy() bool {
	c.RLock()
	defer c.RUnlock()
	return c.clusterHealthy
}

func (c *checker) setHealthy(healthy bool) {
	c.Lock()
	defer c.Unlock()
	c.clusterHealthy = healthy
}
