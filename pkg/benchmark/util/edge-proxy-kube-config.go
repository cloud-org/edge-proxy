/*
Copyright 2022 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package util

import (
	"io/ioutil"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
)

func CreateProxyKubeConfigFile(file string) error {
	c := clientcmdapi.NewConfig()
	cluster := clientcmdapi.NewCluster()
	cluster.Server = "http://127.0.0.1:10261" // 这个地址是 proxy 监听的地址，会反向代理到 server
	c.Clusters["default-cluster"] = cluster

	context := clientcmdapi.NewContext()
	context.Cluster = "default-cluster"
	context.Namespace = "default"
	context.AuthInfo = "default-auth"

	c.Contexts["default-context"] = context
	c.CurrentContext = "default-context"

	data, err := clientcmd.Write(*c)
	if err != nil {
		klog.Errorf("clientcmd.Write error %v", err)
		return err
	}
	return ioutil.WriteFile(file, data, 0111)
}
