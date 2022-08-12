module code.aliyun.com/openyurt/edge-proxy

go 1.16

require (
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/json-iterator/go v1.1.11
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/openyurtio/openyurt v0.7.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/net v0.0.0-20220802222814-0bcc04d9c69b // indirect
	golang.org/x/sys v0.0.0-20220731174439-a90be440212d // indirect
	golang.org/x/tools v0.1.12 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
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
