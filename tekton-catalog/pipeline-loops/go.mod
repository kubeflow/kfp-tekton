module github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops

go 1.13

require (
	github.com/cenkalti/backoff/v4 v4.2.0
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/google/go-cmp v0.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/kubeflow/kfp-tekton/tekton-catalog/cache v0.0.0
	github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore v0.0.0
	github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask v0.0.0-00010101000000-000000000000
	github.com/kubeflow/pipelines/third_party/ml-metadata v0.0.0-20231027040853-58ce09e07d03
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/tektoncd/pipeline v0.53.2
	go.uber.org/zap v1.26.0
	gomodules.xyz/jsonpatch/v2 v2.4.0
	k8s.io/api v0.27.1
	k8s.io/apimachinery v0.27.3
	k8s.io/client-go v0.27.2
	k8s.io/utils v0.0.0-20230505201702-9f6742963106
	knative.dev/pkg v0.0.0-20231011201526-df28feae6d34
)

replace (
	github.com/kubeflow/kfp-tekton/tekton-catalog/cache => ../cache/
	github.com/kubeflow/kfp-tekton/tekton-catalog/objectstore => ../objectstore/
	github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask => ../tekton-kfptask/
	k8s.io/api => k8s.io/api v0.25.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.26.5
	k8s.io/client-go => k8s.io/client-go v0.25.9
	k8s.io/code-generator => k8s.io/code-generator v0.25.9
	k8s.io/kubernetes => k8s.io/kubernetes v1.11.1
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
)
