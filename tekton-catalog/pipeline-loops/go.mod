module github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops

go 1.13

require (
	github.com/docker/cli v0.0.0-20200303162255-7d407207c304 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.18.0
	go.opencensus.io v0.22.4
	go.uber.org/zap v1.15.0
	golang.org/x/tools v0.0.0-20200924205911-8a9a89368bd3 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0
	gotest.tools/v3 v3.0.2 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
)

// Knative deps
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
)

// Pin k8s deps to v0.18.8
replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/apiserver => k8s.io/apiserver v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
