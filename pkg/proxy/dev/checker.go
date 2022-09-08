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
	// remoteServer kube-apiserver url
	remoteServer *url.URL
	// clusterHealthy remote server health or not
	clusterHealthy bool
	// lastTime last health status update time
	lastTime time.Time
}

// NewChecker create a checker with remoteServer
func NewChecker(remoteServer *url.URL) *checker {
	return &checker{
		remoteServer: remoteServer,
		lastTime:     time.Now(),
	}
}

// start start health checker
func (c *checker) start(stopCh <-chan struct{}) {
	c.check() // check once when start
	go c.loop(stopCh)
}

// loop for health check loop
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

// check check remote server is healthy or not
func (c *checker) check() {
	if !health.CheckClusterIsHealthyByGet(c.remoteServer.String()) {
		c.markAsUnhealthy()
		return
	}

	c.markAsHealthy()
}

// markAsHealthy mark a remote server healthy
func (c *checker) markAsHealthy() {
	c.setHealthy(true)
	now := time.Now()
	c.lastTime = now
}

// markAsUnhealthy mark a remote server unhealthy
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

// isHealthy get remote server's health status
func (c *checker) isHealthy() bool {
	c.RLock()
	defer c.RUnlock()
	return c.clusterHealthy
}

// setHealthy set a remote server's health status
func (c *checker) setHealthy(healthy bool) {
	c.Lock()
	defer c.Unlock()
	c.clusterHealthy = healthy
}
