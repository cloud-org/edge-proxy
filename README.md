<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [edge-proxy本地开发](#edge-proxy%E6%9C%AC%E5%9C%B0%E5%BC%80%E5%8F%91)
  - [运行环境](#%E8%BF%90%E8%A1%8C%E7%8E%AF%E5%A2%83)
  - [构建二进制](#%E6%9E%84%E5%BB%BA%E4%BA%8C%E8%BF%9B%E5%88%B6)
  - [local test use minikube not in cluster(use kubeconfig)](#local-test-use-minikube-not-in-clusteruse-kubeconfig)
  - [docker build and push](#docker-build-and-push)
  - [如何本地测试](#%E5%A6%82%E4%BD%95%E6%9C%AC%E5%9C%B0%E6%B5%8B%E8%AF%95)
    - [申请共享测试集群的kubeconfig 文件](#%E7%94%B3%E8%AF%B7%E5%85%B1%E4%BA%AB%E6%B5%8B%E8%AF%95%E9%9B%86%E7%BE%A4%E7%9A%84kubeconfig-%E6%96%87%E4%BB%B6)
    - [本地安装kubectl](#%E6%9C%AC%E5%9C%B0%E5%AE%89%E8%A3%85kubectl)
    - [构建docker镜像，生成测试用的manifest文件](#%E6%9E%84%E5%BB%BAdocker%E9%95%9C%E5%83%8F%E7%94%9F%E6%88%90%E6%B5%8B%E8%AF%95%E7%94%A8%E7%9A%84manifest%E6%96%87%E4%BB%B6)
    - [创建manifest资源](#%E5%88%9B%E5%BB%BAmanifest%E8%B5%84%E6%BA%90)
    - [重复测试流程](#%E9%87%8D%E5%A4%8D%E6%B5%8B%E8%AF%95%E6%B5%81%E7%A8%8B)
    - [其他说明](#%E5%85%B6%E4%BB%96%E8%AF%B4%E6%98%8E)
- [edge-proxy实现指南](#edge-proxy%E5%AE%9E%E7%8E%B0%E6%8C%87%E5%8D%97)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# edge-proxy本地开发

## 运行环境

linux 运行环境


## 构建二进制

```
# make build
```

make build 命令会生成两个二进制:edge-proxy 和 benchmark, 存放在目录：_output/local/bin/{GOOS}/{GOARCH}/

* edge-proxy 是本次比赛的框架代码，选手可以根据里面的主体逻辑实现对应的功能。
* benchmark 是提供的一个用于本地调试edge-proxy 功能的工具，选手也参照benchmark 提供的代码，对edge-proxy更详细的测试。

## local test use minikube not in cluster(use kubeconfig)

```sh
export server_addr=$(kubectl config view --minify -o=jsonpath="{.clusters[*].cluster.server}")
./edge-proxy --server-addr ${server_addr} --use-kubeconfig true --enable-sample-handler true
./benchmark --namespace default --use-kubeconfig
```

## docker build and push

````sh
make docker-build IMAGE_REPO=registry.cn-shanghai.aliyuncs.com/cloud-native-edge-proxy IMAGE_TAG=v0.0.1 REGION=cn
````

## 如何本地测试

### 申请共享测试集群的kubeconfig 文件

* 加入【赛道2】2022云原生编程挑战赛选手钉钉群：44745334 
* 艾特 群管理员 张杰 申请 kubeconfig 文件， 申请内容格式如下：

```
	云原生挑战赛边缘赛道申请kubeconfig 文件,参赛队伍名称: ****
```
*  管理员会私信发送对应kubeconfig 文件， 此kubeconfig 文件只有namespace 下的pod ,configmap 等操作权限，足够参赛选手测试 
*  将收到的kubeconfig文件内容保存到本地计算机的`$HOME/.kube/cloudnative-challenge/config` 文件里, 切记一定要执行这一步， 否则后续的镜像构建这一步会出错。

### 本地安装kubectl 

[安装文档](https://kubernetes.io/zh-cn/docs/tasks/tools/install-kubectl-linux/)

版本要求： v1.20.11

后续的调试请使用 `kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config ` 命令进行测试， 例如：

```
kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config get pod

```

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
`manifest.yaml` 文件里主要包括了测试的pod资源对象。

### 创建manifest资源

`kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config apply -f _output/manifest.yaml`

执行后， 会在对应命名空间下生成名字为benchmark 的pod对象，此pod包含了两个容器， 一个是edge-proxy 的容器， 一个是benchmark 的容器， 可以使用kubectl命令查看:

`kubectl  --kubeconfig=$HOME/.kube/cloudnative-challenge/config get pod benchmark -o yaml`

可以通过 `kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config logs -f ` 命令查看pod 里两个容器产生的日志，进而进行问题的排查：

```
#查看bench-mark 容器日志: 
kubectl  --kubeconfig=$HOME/.kube/cloudnative-challenge/config logs -f benchmark bench-mark

# 查看edge-proxy 容器日志:
kubectl  --kubeconfig=$HOME/.kube/cloudnative-challenge/config logs -f benchmark edge-proxy
```

### 重复测试流程
1.  修改edge-proxy 代码逻辑
2.  执行 `make docker-build` 命令， 重新构建镜像，并push 镜像
3.  删掉测试pod 
```
kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config delete -f _output/manifest.yaml
```
4.  重新创建测试pod 
```
kubectl --kubeconfig=$HOME/.kube/cloudnative-challenge/config apply -f _output/manifest.yaml
```

### 其他说明

参赛选手可以修改和完善benchmark 的代码逻辑，实现更全面的功能测试，目的是实现edge-proxy 的所有功能， 这样在大赛官网上提交后，会有更高的测试通过命中率。


# edge-proxy实现指南

- 参照pkg/sample/handler.go 实现`HandlerFactory`接口(pkg/proxy/interface.go)，确保实现的handler可以完成下述功能:
  1. 透明代理能力: 基于`EdgeProxyConfiguration.RemoteServers[0]`和`EdgeProxyConfiguration.RT`转发请求到指定服务器，同时服务器的response返回给请求方
  2. 数据过滤能力: 过滤response数据(如Kubernetes的Pod,ConfigMap)中Object.Name={skip-xxx}条件的数据
  3. 数据缓存能力: 基于`EdgeProxyConfiguration.SerializerManager`和`EdgeProxyConfiguration.DiskCachePath`把服务器的response缓存到本地

- 实现完成的handler，需要在cmd/edge-proxy/main.go 中import，确保handler中的init()可以正常初始化。
  > 可以参考sample handler的引用：_ "code.aliyun.com/openyurt/edge-proxy/pkg/proxy/sample"
