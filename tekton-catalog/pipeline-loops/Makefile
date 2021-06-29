BIN_DIR ?= _output/bin

DOCKER_REGISTRY ?= aipipeline

TIMEOUT_TESTS = 2.5m

SYSTEM_NAMESPACE=tekton-pipelines


# If tag not explicitly set in users default to the git sha.
TAG ?= v0.0.1

.EXPORT_ALL_VARIABLES:

all:
	$(info try make help)

KO = ko
ko: PACKAGE=github.com/google/ko/cmd/ko

apply: $(info apply PipelineLoop controller to the k8s cluster )
	$(KO) apply -f config

test-all:  $(info test-all: run unit tests )
	go test -v -cover -timeout=${TIMEOUT_TESTS} ./...

init:
	mkdir -p ${BIN_DIR}

cli: init $(info cli: build validation cli )
	go build -o=${BIN_DIR}/pipelineloop-cli ./cmd/cli

local: init
	go build -o=${BIN_DIR}/pipelineloop-controller ./cmd/controller
	go build -o=${BIN_DIR}/pipelineloop-webhook ./cmd/webhook
	go build -o=${BIN_DIR}/pipelineloop-cli ./cmd/cli

build-linux: init
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=${BIN_DIR}/pipelineloop-controller ./cmd/controller
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=${BIN_DIR}/pipelineloop-webhook ./cmd/webhook
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=${BIN_DIR}/pipelineloop-cli ./cmd/cli

images: build-linux
	docker build --build-arg bin_name=pipelineloop-controller . -t ${DOCKER_REGISTRY}/pipelineloop-controller:$(TAG)
	docker build --build-arg bin_name=pipelineloop-webhook . -t ${DOCKER_REGISTRY}/pipelineloop-webhook:$(TAG)

update:
	go mod download
	go mod tidy
	go mod vendor

clean:
	rm -r ${BIN_DIR}
	rm -f *.log
