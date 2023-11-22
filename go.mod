module github.com/kubeflow/pipelines

require (
	github.com/Masterminds/squirrel v0.0.0-20190107164353-fa735ea14f09
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/denisenkom/go-mssqldb v0.9.0 // indirect
	github.com/eapache/go-resiliency v1.2.0
	github.com/fsnotify/fsnotify v1.6.0
	github.com/go-openapi/errors v0.20.2
	github.com/go-openapi/runtime v0.21.1
	github.com/go-openapi/strfmt v0.21.1
	github.com/go-openapi/swag v0.22.3
	github.com/go-openapi/validate v0.20.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/glog v1.1.0
	github.com/golang/protobuf v1.5.3
	github.com/google/addlicense v0.0.0-20200906110928-a0294312aa76
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3
	github.com/jinzhu/gorm v1.9.12
	github.com/kubeflow/pipelines/api v0.0.0-20211026071850-2e3fb5efff56
	github.com/lestrrat-go/strftime v1.0.4
	github.com/mattn/go-sqlite3 v2.0.1+incompatible
	github.com/minio/minio-go/v6 v6.0.57
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.15.1
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.8.4
	github.com/tektoncd/pipeline v0.53.2
	github.com/tidwall/pretty v1.1.0 // indirect
	go.uber.org/zap v1.26.0
	golang.org/x/net v0.17.0
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97
	google.golang.org/grpc v1.58.3
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.31.0
	k8s.io/api v0.27.2
	k8s.io/apimachinery v0.27.3
	k8s.io/client-go v0.27.2
	k8s.io/code-generator v0.27.2
	sigs.k8s.io/controller-runtime v0.15.0
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.9.0
	github.com/prometheus/common => github.com/prometheus/common v0.26.0
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.4.4
	go.opencensus.io => go.opencensus.io v0.22.5
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

go 1.13
