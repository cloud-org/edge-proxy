package config

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInitClient(t *testing.T) {
	got, err := InitClient(true)
	if err != nil {
		t.Error(err)
		return
	}

	resp, err := got.CoreV1().ConfigMaps("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Error(err)
		return
	}

	for i := 0; i < len(resp.Items); i++ {
		item := resp.Items[i]
		// item.ObjectMeta 是嵌入结构 所以 item.Name 和 item.ObjectMeta.Name 是一致的
		// type ConfigMap struct {
		//    v1.TypeMeta   `json:",inline"`
		//    v1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
		//}
		t.Logf("item.Name: %s\n", item.Name)
	}
}
