<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [edge-proxy](#edge-proxy)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### edge-proxy

return resourceusage cache 并发安全分支，解决了类似多协程并发请求存在缓存击穿问题。

- 关键函数

```go
//returnCacheResourceUsage if labelSelector contains type=resourceusage, then return mem data if ok
func (d *devFactory) returnCacheResourceUsage(handler http.Handler) http.Handler {
	var count int32
	//var countLock sync.Mutex
	var resourceLock sync.Mutex

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// if resource usage cache, then return, else continue
		labelSelector := req.URL.Query().Get("labelSelector") // filter then enter
		defer func() {
			label := labelSelector
			if strings.Contains(label, resourceLabel) {
				atomic.AddInt32(&count, 1)
				klog.V(5).Infof("latest count %v", atomic.LoadInt32(&count))
			}
		}()
		if d.getResourceCache() && strings.Contains(labelSelector, resourceLabel) {
			//klog.Infof("return resource cache")
			klog.V(5).Infof("enter get resource cache")
			res, ok := d.cacheMgr.QueryCacheMem("configmaps", d.resourceNs, resourceType)
			if !ok {
				klog.Errorf("may be not resource cache")
				goto end
			}

			rw.Header().Set("Content-Type", "application/json")
			_, err := rw.Write(res)
			if err != nil {
				klog.Errorf("rw.Write err: %v", err)
				goto end
			}
			//rw.WriteHeader(http.StatusOK)
			// return if not err
			return
		}
	end:

		info, err := d.resolver.NewRequestInfo(req)
		if err != nil {
			klog.Errorf("resolver request info err: %v", err)
			return
		}
		// inject info
		req = req.WithContext(apirequest.WithRequestInfo(req.Context(), info))
		// 全局阻塞
		if checkLabel(info, labelSelector, resourceLabel) {
			resourceLock.Lock()
			defer resourceLock.Unlock()
			// 重新检测 resource cache
			if d.getResourceCache() {
				klog.V(5).Infof("enter after lock check")
			retry:
				res, ok := d.cacheMgr.QueryCacheMem("configmaps", d.resourceNs, resourceType)
				if !ok {
					klog.Errorf("may be not resource cache")
					time.Sleep(10 * time.Millisecond)
					goto retry
				}

				rw.Header().Set("Content-Type", "application/json")
				_, err = rw.Write(res)
				if err != nil {
					klog.Errorf("rw.Write err: %v", err)
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				// 成功才写入 ok header
				//rw.WriteHeader(http.StatusOK)
				// return if not err
				return
			}
			klog.Infof("enter first resource usage")
			// no resource cache
			handler.ServeHTTP(rw, req)
			d.setResourceCache(info.Namespace)
			return
		}
		// other request 其他请求
		handler.ServeHTTP(rw, req)
		return
	})
}

```