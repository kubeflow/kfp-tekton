BIN_DIR ?= _output/bin

DOCKER_REGISTRY ?= aipipeline

# If tag not explicitly set in users default to the git sha.
TAG ?= v0.0.1

.EXPORT_ALL_VARIABLES:

all: local

init:
	mkdir -p ${BIN_DIR}

local: init
	go build -o=${BIN_DIR}/pipelineloop-controller ./cmd/controller
	go build -o=${BIN_DIR}/pipelineloop-webhook ./cmd/webhook

build-linux: init
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=${BIN_DIR}/pipelineloop-controller ./cmd/controller
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=${BIN_DIR}/pipelineloop-webhook ./cmd/webhook

images: build-linux
	docker build --build-arg bin_name=pipelineloop-controller . -t ${DOCKER_REGISTRY}/pipelineloop-controller:$(TAG)
	docker build --build-arg bin_name=pipelineloop-webhook . -t ${DOCKER_REGISTRY}/pipelineloop-webhook:$(TAG)

update:
	go mod download
	go mod tidy
	go mod vendor

clean:
	rm -rf ${BIN_DIR}
	rm -f *.log
