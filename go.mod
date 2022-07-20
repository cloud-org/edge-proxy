module code.aliyun.com/openyurt/edge-proxy

go 1.16

require (
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/imroc/req/v3 v3.14.0
	github.com/openyurtio/openyurt v0.7.0 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/apiserver v0.22.3
	k8s.io/client-go v0.22.3
	k8s.io/klog/v2 v2.9.0
)

replace (
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/apiserver-network-proxy => github.com/openyurtio/apiserver-network-proxy v1.18.8
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22
)
