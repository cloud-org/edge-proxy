package health

import (
	"testing"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/config"
)

func TestCheckClusterIsHealthy(t *testing.T) {

	client, err := config.InitClient(true)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("isHealthy: %v", CheckClusterIsHealthy(client))
}

func TestCheckClusterIsHealthyByGet(t *testing.T) {
	t.Logf("isHealthy: %v", CheckClusterIsHealthyByGet("https://47.102.205.13:6443"))
}
