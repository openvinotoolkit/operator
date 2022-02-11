// module github.com/operator-framework/operator-sdk
module github.com/openvinotoolkit/openshift_operator

go 1.16

require (
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-resty/resty/v2 v2.7.0
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/operator-framework/api v0.8.2-0.20210526151024-41d37db9141f // indirect
	github.com/operator-framework/operator-lib v0.4.1
	github.com/operator-framework/operator-registry v1.15.3
	github.com/prometheus/client_golang v1.7.1
	github.com/sergi/go-diff v1.1.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20201012173705-84dcc777aaee // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20211029224645-99673261e6eb // indirect
	golang.org/x/sys v0.0.0-20210521090106-6ca3eb03dfc2 // indirect
	golang.org/x/tools v0.1.1 // indirect
	gomodules.xyz/jsonpatch/v3 v3.0.1
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/kubectl v0.20.2 // indirect
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	// Using containerd 1.4.0+ resolves an issue with invalid error logging
	// from an init function in containerd. This replace can be removed when
	// one of our direct dependencies begins using containerd v1.4.0+
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.3
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
	sigs.k8s.io/kubebuilder/v3 => sigs.k8s.io/kubebuilder/v3 v3.0.0-alpha.0.0.20210518234629-191170994550
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
