# edge-proxy本地开发

```
// 生成edge-proxy二进制, 存放在_output/local/bin/{GOOS}/{GOARCH}/edge-proxy
#$ make build

// 生成edge-proxy镜像, 通过IMAGE_REPO指定镜像repo, IMAGE_TAG指定镜像tag, REGION指定是否需要代理
// 比如: 生成镜像openyurt/edge-proxy:v1.0，并且使用golang代理
#$ make docker-build IMAGE_REPO=openyurt IMAGE_TAG=v1.0 REGION=cn
```

# edge-proxy实现指南

参照pkg/sample/handler.go 实现`HandlerFactory`接口(pkg/proxy/interface.go)，确保实现的handler可以完成下述功能:
1. 透明代理能力: 基于`EdgeProxyConfiguration.RemoteServers[0]`和`EdgeProxyConfiguration.RT`转发请求到指定服务器，同时服务器的response返回给请求方
2. 数据过滤能力: 过滤返回数据(如Kubernetes的Pod,ConfigMap)中Object.Name={skip-xxx}条件的数据
3. 数据缓存能力: 基于`EdgeProxyConfiguration.SerializerManager`和`EdgeProxyConfiguration.DishCachePath`把服务器的response缓存到本地
