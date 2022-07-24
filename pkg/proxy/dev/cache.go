package dev

import (
	"sync"
)

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
