# edge-proxy本地开发

## 构建二进制
```
# make build
```
make build 命令会生成两个二进制:edge-proxy 和 benchmark, 存放在目录：_output/local/bin/{GOOS}/{GOARCH}/

* edge-proxy 是本次比赛的框架代码，选手可以根据里面的主体逻辑实现对应的功能。
*  benchmark 是提供的一个用于本地调试edge-proxy 功能的工具，选手也参照benchmark 提供的代码，对edge-proxy更详细的测试。

## 如何本地测试

### 创建边托管集群

[创建说明文档](https://help.aliyun.com/document_detail/124723.html)

*说明* 
* 集群配置：地域选择华东2 上海，勾选使用EIP暴露API Server
* Worker配置 ，ECS 配置至少4C 4G，数量选择1 ,并设置登录密码

等待大约10分钟集群创建完后， 点击进入对应的集群，点击左侧导航栏集群信息， 看到集群的概览信息。
点击连接信息TAB 页，复制公网访问里黑色框的内容， 并保存到本地计算机的`$HOME/.kube/config` 文件里, 切记一定要执行这一步， 否则后续的镜像构建这一步会出错。

### 本地安装kubectl 

[安装文档](https://kubernetes.feisky.xyz/setup/kubectl)
安装完  kubectl  工具后， 即可以在本地通过kubectl 命令操作ACK@Edge集群 

### 构建docker镜像，生成测试用的manifest文件

`make docker-build` 命令用于生成并push edge-proxy镜像 和并且产生用于本地测试的manifest 文件

参数解析：

*  IMAGE_REPO指定镜像repo
*  IMAGE_TAG指定镜像tag
*  REGION指定是否需要代理
*  DOCKER_USERNAME 指定阿里云镜像仓库的用户名
*  DOCKER_PASSWD 指定阿里云镜像仓库的密码

// 比如: 生成镜像registry.cn-shanghai.aliyuncs.com/cloudnative-challenge/edge-proxy:v1.0，并且使用golang代理， 命令如下:

```
#$ make docker-build IMAGE_REPO=registry.cn-shanghai.aliyuncs.com/cloudnative-challenge IMAGE_TAG=v1.0 REGION=cn DOCKER_USERNAME=** DOCKER_PASSWD=**
```

若 `make docker-build` 命令执行成功， 会自动push 镜像到对应的阿里云镜像仓库中，并且在`_output/` 目录下生成 `manifest.yaml` 文件。
`manifest.yaml` 文件里主要包括了一些clusterrole, clusterrolebinding, secret, deployment 资源对象。

### 运行manifest 文件

`kubectl apply -f _output/manifest.yaml`

执行后， 会在default 命名空间下生成名字为benchmark 的deployment 对象， 此资源对象产生的pod包含了两个容器， 一个是edge-proxy 的容器， 一个是benchmark 的容器

`kubectl get deployments.apps benchmark -o yaml`

可以通过 `kubectl logs -f ` 命令查看pod 里两个容器产生的日志，进而进行问题的排查。


### 其他说明

参赛选手可以修改和完善benchmark 的代码逻辑，实现更全面的功能测试，这样在大赛官网上提交后，会有更高的测试通过命中率。


# edge-proxy实现指南

- 参照pkg/sample/handler.go 实现`HandlerFactory`接口(pkg/proxy/interface.go)，确保实现的handler可以完成下述功能:
  1. 透明代理能力: 基于`EdgeProxyConfiguration.RemoteServers[0]`和`EdgeProxyConfiguration.RT`转发请求到指定服务器，同时服务器的response返回给请求方
  2. 数据过滤能力: 过滤response数据(如Kubernetes的Pod,ConfigMap)中Object.Name={skip-xxx}条件的数据
  3. 数据缓存能力: 基于`EdgeProxyConfiguration.SerializerManager`和`EdgeProxyConfiguration.DishCachePath`把服务器的response缓存到本地

- 实现完成的handler，需要在cmd/edge-proxy/main.go 中import，确保handler中的init()可以正常初始化。
  > 可以参考sample handler的引用：_ "code.aliyun.com/openyurt/edge-proxy/pkg/proxy/sample"
