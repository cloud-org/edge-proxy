package config

import (
	"io/ioutil"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeConfigPath = os.Getenv("HOME") + "/.kube/config"

//InitClient 初始化 k8s 客户端
func InitClient(usekubeconfig bool) (*kubernetes.Clientset, error) {
	var cfg *rest.Config
	var err error
	if usekubeconfig {
		cfg, err = GetRestConf()
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	// 生成 clientset 配置
	return kubernetes.NewForConfig(cfg)
}

//GetRestConf 获取 k8s restful client 配置
func GetRestConf() (*rest.Config, error) {
	// 读 kubeconfig 文件
	kubeconfig, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	// 生成 rest client 配置
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}
