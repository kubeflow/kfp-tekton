module github.com/vincent-pli/kubectl-wrapper

go 1.13

require (
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/docker/docker v1.4.2-0.20190327010347-be7ac8be2ae0 // indirect
	github.com/google/martian v2.1.0+incompatible
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/common v0.7.0 // indirect
	github.com/prometheus/procfs v0.0.7 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/tidwall/gjson v1.6.0
	//github.ibm.com/panpxpx/klsf v0.0.0-00010101000000-000000000000
	go.etcd.io/etcd v0.5.0-alpha.5.0.20191023171146-3cf2f69b5738 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/sys v0.0.0-20191115151921-52ab43148777 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20191116214431-80313e1ba718 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191115221424-83cc0476cb11 // indirect
	google.golang.org/grpc v1.25.1 // indirect
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.17.0
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
	sigs.k8s.io/controller-runtime v0.5.1
)

replace (
	//github.ibm.com/panpxpx/klsf => /Users/pengli/go/src/github.ibm.com/panpxpx/klsf
	k8s.io/api => k8s.io/api v0.0.0-20191122220107-b5267f2975e0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191122222427-64482ea217ff
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191121175448-79c2a76c473a
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191121180716-5a28f8b2ad8e
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191122222818-9150eb3ded31
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191121175918-3a262fe58afa
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191121022508-6371aabbd7a7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191122223827-289de4a64c1c
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191121015212-c4c8f8345c7e
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191122163614-46ba8a4433be
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20191121183020-775aa3c1cf73
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191121022617-4b18d293964d
	k8s.io/gengo => k8s.io/gengo v0.0.0-20191120174120-e74f70b9b27e
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191122221605-1e8d331e4dcc
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191122223648-5cfd5067047c
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191122223145-16f2c0c680a0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191121022142-fe73241eced9
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191122225023-1e3c8b70f494
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191122223325-9316382755ad
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191122224431-860df69ff5cc
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191122222628-19ed227de2b6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191122221846-294c70c3d5d4
	k8s.io/utils => k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
)
