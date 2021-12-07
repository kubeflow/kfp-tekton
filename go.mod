module github.com/kubeflow/pipelines

require (
	github.com/Masterminds/squirrel v0.0.0-20190107164353-fa735ea14f09
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-ini/ini v1.62.0 // indirect
	github.com/go-openapi/errors v0.19.9
	github.com/go-openapi/runtime v0.19.24
	github.com/go-openapi/strfmt v0.19.11
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.1
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/google/addlicense v0.0.0-20200906110928-a0294312aa76
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/jinzhu/gorm v1.9.12
	github.com/lestrrat-go/strftime v1.0.4
	github.com/mattn/go-sqlite3 v2.0.1+incompatible
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.30.0
	go.uber.org/zap v1.19.1
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	google.golang.org/genproto v0.0.0-20211016002631-37fc39342514
	google.golang.org/grpc v1.41.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.21.5
	k8s.io/apimachinery v0.21.5
	k8s.io/client-go v0.21.5
	k8s.io/code-generator v0.21.5
	sigs.k8s.io/controller-runtime v0.9.7
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.55.0
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	// TODO: remove temporary patch
	github.com/kubeflow/pipelines/api => ./api
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.9.0
	github.com/prometheus/common => github.com/prometheus/common v0.15.0
	go.opencensus.io => go.opencensus.io v0.22.5
	k8s.io/api => k8s.io/api v0.21.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.5
	k8s.io/apiserver => k8s.io/apiserver v0.21.5
	k8s.io/client-go => k8s.io/client-go v0.21.5
	k8s.io/code-generator => k8s.io/code-generator v0.21.5
)

go 1.13
