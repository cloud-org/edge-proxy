package dev

import (
	"sync"
)

// cacheChecker 标注是否已经可以进行 cache，原本是保存一致性结果的时候会用到，不过现在不用这个组件了。
type cacheChecker struct {
	sync.RWMutex
	check bool
}

func NewCacheChecker() *cacheChecker {
	return &cacheChecker{
		check: false,
	}
}

func (c *cacheChecker) CanCache() bool {
	c.RLock()
	defer c.RUnlock()
	return c.check
}

func (c *cacheChecker) SetCanCache() {
	if !c.CanCache() {
		c.Lock()
		c.check = true
		c.Unlock()
	}
}
