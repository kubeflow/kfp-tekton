module github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops

go 1.13

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.5 // indirect
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.27.1
	go.uber.org/zap v1.18.1
	gomodules.xyz/jsonpatch/v2 v2.2.0
	gopkg.in/evanphx/json-patch.v4 v4.9.0 // indirect
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
)
