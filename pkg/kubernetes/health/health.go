package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func CheckClusterIsHealthyByGet(url string) bool {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: tr,
	}

	apiUrl := fmt.Sprintf("%s/livez", url)
	//klog.Infof("apiUrl: %v", apiUrl)
	resp, err := client.Get(apiUrl)
	if err != nil {
		klog.Errorf("get %s err: %v", apiUrl, err)
		return false
	}

	defer resp.Body.Close()

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("readAll err: %v", err)
		return false
	}

	klog.Infof("check livez content: %v", string(res))
	if string(res) != "ok" {
		return false
	}

	return true
}

func CheckClusterIsHealthy(client *kubernetes.Clientset) bool {

	path := "/livez"

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()
	content, err := client.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		klog.Errorf("check livez err: %v", err)
		return false
	}

	res := string(content)
	klog.Infof("check livez content: %v", res)
	if res != "ok" {
		return false
	}

	return true
}
