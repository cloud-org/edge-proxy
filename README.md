<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [edge-proxy本地开发](#edge-proxy%E6%9C%AC%E5%9C%B0%E5%BC%80%E5%8F%91)
  - [运行环境](#%E8%BF%90%E8%A1%8C%E7%8E%AF%E5%A2%83)
  - [构建二进制](#%E6%9E%84%E5%BB%BA%E4%BA%8C%E8%BF%9B%E5%88%B6)
  - [local test use minikube not in cluster(use kubeconfig)](#local-test-use-minikube-not-in-clusteruse-kubeconfig)
  - [docker build and push](#docker-build-and-push)
  - [如何本地测试](#%E5%A6%82%E4%BD%95%E6%9C%AC%E5%9C%B0%E6%B5%8B%E8%AF%95)
    - [构建docker镜像，生成测试用的 manifest 文件](#%E6%9E%84%E5%BB%BAdocker%E9%95%9C%E5%83%8F%E7%94%9F%E6%88%90%E6%B5%8B%E8%AF%95%E7%94%A8%E7%9A%84-manifest-%E6%96%87%E4%BB%B6)
    - [创建 manifest 资源](#%E5%88%9B%E5%BB%BA-manifest-%E8%B5%84%E6%BA%90)
    - [重复测试流程](#%E9%87%8D%E5%A4%8D%E6%B5%8B%E8%AF%95%E6%B5%81%E7%A8%8B)
  - [Coding Time](#coding-time)

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
* benchmark 是提供的一个用于本地调试 edge-proxy 功能的工具，选手也参照 benchmark 提供的代码，对 edge-proxy 更详细的测试。

## local test use minikube not in cluster(use kubeconfig)

```sh
export server_addr=$(kubectl config view --minify -o=jsonpath="{.clusters[*].cluster.server}")
export ns=$(kubectl get cm kube-root-ca.crt -o=jsonpath="{.metadata.namespace}")
./edge-proxy --server-addr ${server_addr} --use-kubeconfig true --enable-sample-handler true --disk-cache-path ~/.kube/cloudnative-challenge/cache
./benchmark --namespace ${ns} --use-kubeconfig
```

## docker build and push

````sh
make docker-build IMAGE_REPO=registry.cn-shanghai.aliyuncs.com/cloud-native-edge-proxy IMAGE_TAG=v0.0.1 REGION=cn
````

## 如何本地测试

```sh
alias kubectl='kubectl --kubeconfig ~/.kube/cloudnative-challenge/config'
```

### 构建docker镜像，生成测试用的 manifest 文件

`make docker-build` 命令用于生成并push edge-proxy镜像 和并且产生用于本地测试的manifest 文件

参数解析：

* IMAGE_REPO指定镜像 repo
* IMAGE_TAG指定镜像 tag
* REGION指定是否需要代理
* DOCKER_USERNAME 指定阿里云镜像仓库的用户名
* DOCKER_PASSWD 指定阿里云镜像仓库的密码

// 比如: 生成镜像registry.cn-shanghai.aliyuncs.com/cloudnative-challenge/edge-proxy:v1.0，并且使用golang代理， 命令如下:

```
$ make docker-build IMAGE_REPO=registry.cn-shanghai.aliyuncs.com/cloudnative-challenge IMAGE_TAG=v1.0 REGION=cn DOCKER_USERNAME=** DOCKER_PASSWD=**
```

若 `make docker-build` 命令执行成功， 会自动push 镜像到对应的阿里云镜像仓库中，并且在`_output/` 目录下生成 `manifest.yaml` 文件。
`manifest.yaml` 文件里主要包括了测试的pod资源对象。

### 创建 manifest 资源

`kubectl apply -f _output/manifest.yaml`

执行后， 会在对应命名空间下生成名字为benchmark 的pod对象，此pod包含了两个容器， 一个是edge-proxy 的容器， 一个是benchmark 的容器， 可以使用kubectl命令查看:

`kubectl get pod benchmark -o yaml`

```
#查看 bench-mark 容器日志: 
kubectl logs -f benchmark bench-mark

# 查看 edge-proxy 容器日志:
kubectl logs -f benchmark edge-proxy
```

### 重复测试流程

1. 修改 edge-proxy 代码逻辑
2. 执行 `make docker-build` 命令， 重新构建镜像，并 push 镜像
3. 删掉测试pod

```
kubectl delete -f _output/manifest.yaml
```

4. 重新创建测试 pod

```
kubectl apply -f _output/manifest.yaml
```

## Coding Time

![wakatime](https://wakatime.com/badge/user/01c864c3-99e2-47a2-ad28-cc0f36b02f39/project/8ae39e6c-e2ff-45a3-acbf-e5deee6bdfa8.svg)
