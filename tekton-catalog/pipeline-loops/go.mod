module github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops

go 1.13

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jinzhu/gorm v1.9.16 // indirect
	github.com/kubeflow/kfp-tekton/tekton-catalog/cache v0.0.0
	github.com/kubeflow/kfp-tekton/tekton-catalog/objectstorelogger v0.0.0
	github.com/mattn/go-sqlite3 v1.14.0 // indirect
	github.com/tektoncd/pipeline v0.30.0
	go.uber.org/zap v1.19.1
	gomodules.xyz/jsonpatch/v2 v2.2.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	knative.dev/pkg v0.0.0-20211101212339-96c0204a70dc
)

replace (
	github.com/kubeflow/kfp-tekton/tekton-catalog/cache => ../cache/
	github.com/kubeflow/kfp-tekton/tekton-catalog/objectstorelogger => ../objectstorelogger/
)
