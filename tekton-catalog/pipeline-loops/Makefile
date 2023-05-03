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

apply: $(info apply: Apply PipelineLoop controller to the k8s cluster )
	$(KO) apply -f config

test-all: update init $(info test-all: run unit tests )
	go test -v -cover -timeout=${TIMEOUT_TESTS} ./...

.PHONY: init
init:
	mkdir -p ${BIN_DIR}

.PHONY: cli
cli: update init $(info cli: build validation cli )
	go build -o=${BIN_DIR}/pipelineloop-cli ./cmd/cli

.PHONY: validate-testdata-python-sdk
validate-testdata-python-sdk: cli $(info validate-testdata-python-sdk: validate testdata for python SDK )
## works w/o findutils installed, but globbing has a limit
## for x in ../../sdk/python/tests/compiler/testdata/*yaml; do ${BIN_DIR}/pipelineloop-cli -f $$x || true ; done
	@find ../../sdk/python/tests/compiler/testdata \
		\( -type f -name "*yaml" -not -path "*\test_data\*" -not -name "*component.yaml" \) \
		-print0 | xargs -0 -n1 ${BIN_DIR}/pipelineloop-cli -f

.PHONY: validate-examples
validate-examples: cli $(info validate-examples: validate example yamls for pipelineloop custom task. )
	@find ./examples/ \
		\( -type f -name "*yaml" \) -print0 | xargs -0 -n1 ${BIN_DIR}/pipelineloop-cli -f

local: update init
	go build -o=${BIN_DIR}/pipelineloop-controller ./cmd/controller
	go build -o=${BIN_DIR}/pipelineloop-webhook ./cmd/webhook
	go build -o=${BIN_DIR}/pipelineloop-cli ./cmd/cli

build-linux: update init
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
