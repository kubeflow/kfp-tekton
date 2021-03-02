# Copyright 2020-2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Acknowledgements:
#  - The help target was derived from https://stackoverflow.com/a/35730328/5601796

VENV ?= .venv
KFP_TEKTON_RELEASE ?= v0.7.0
export VIRTUAL_ENV := $(abspath ${VENV})
export PATH := ${VIRTUAL_ENV}/bin:${PATH}
DOCKER_REGISTRY ?= aipipeline

.PHONY: help
help: ## Display the Make targets
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

.PHONY: venv
venv: $(VENV)/bin/activate ## Create and activate virtual environment
$(VENV)/bin/activate: sdk/python/setup.py
# create/update the VENV when there was a change to setup.py
# check if kfp-tekton is already installed (Travis/CI did during install step)
# use pip from the specified VENV as opposed to any pip available in the shell
	@echo "VENV=$(VENV)"
	@test -d $(VENV) || python3 -m venv $(VENV)
	@$(VENV)/bin/pip show kfp-tekton >/dev/null 2>&1 || $(VENV)/bin/pip install -e sdk/python
	@touch $(VENV)/bin/activate

.PHONY: install
install: venv ## Install the kfp_tekton compiler in a virtual environment
	@echo "Run 'source $(VENV)/bin/activate' to activate the virtual environment."

.PHONY: unit_test
unit_test: venv ## Run compiler unit tests
	@echo "=================================================================="
	@echo "Optional environment variables to configure $@, examples:"
	@sed -n -e 's/# *\(make $@ .*\)/  \1/p' sdk/python/tests/compiler/compiler_tests.py
	@echo "=================================================================="
	@pip show pytest > /dev/null 2>&1 || pip install pytest
	@sdk/python/tests/run_tests.sh
	@echo "$@: OK"

.PHONY: e2e_test
e2e_test: venv ## Run compiler end-to-end tests (requires kubectl and tkn CLI)
	@echo "=================================================================="
	@echo "Optional environment variables to configure $@, examples:"
	@sed -n -e 's/# *\(make $@ .*\)/  \1/p' sdk/python/tests/compiler/compiler_tests_e2e.py
	@echo "=================================================================="
	@which kubectl > /dev/null || (echo "Missing kubectl CLI" && exit 1)
	@test -z "${KUBECONFIG}" && echo "KUBECONFIG not set" || echo "KUBECONFIG=${KUBECONFIG}"
	@kubectl version --short || (echo "Failed to access kubernetes cluster" && exit 1)
	@which tkn > /dev/null || (echo "Missing tkn CLI" && exit 1)
	@sdk/python/tests/run_e2e_tests.sh
	@echo "$@: OK"

.PHONY: test
test: unit_test e2e_test ## Run compiler unit tests and end-to-end tests
	@echo "$@: OK"

.PHONY: report
report: ## Report compilation status of KFP testdata DSL scripts
	@sdk/python/tests/test_kfp_samples.sh
	@echo "$@: OK"

.PHONY: lint
lint: venv ## Check Python code style compliance
	@which flake8 > /dev/null || pip install flake8
	@flake8 sdk/python --show-source --statistics \
		--select=E9,E2,E3,E5,F63,F7,F82,F4,F841,W291,W292 \
		--per-file-ignores sdk/python/tests/compiler/testdata/*:F841 \
 		--max-line-length=140
	@echo "$@: OK"

.PHONY: check_license
check_license: ## Check for license header in source files
	@find ./sdk/python -type f \( -name '*.py' -o -name '*.yaml' \) -exec \
		grep -H -E -o -c 'Copyright 20.* kubeflow.org' {} \; | \
		grep -E ':0$$' | sed 's/..$$//' | \
		grep . && echo "The files listed above are missing the license header" && exit 1 || \
		echo "$@: OK"

.PHONY: check_mdtoc
check_mdtoc: ## Check Markdown files for valid the Table of Contents
	@find guides samples sdk *.md -type f -name '*.md' -exec \
		grep -l -i 'Table of Contents' {} \; | sort | \
		while read -r md_file; do \
			grep -oE '^ *[-+*] \[[^]]+\]\(#[^)]+\)' "$${md_file}" |  sed -e 's/[-+*] /- /g' > md_file_toc; \
			./tools/mdtoc.sh "$${md_file}" > generated_toc; \
			diff -w md_file_toc generated_toc || echo "$${md_file}"; \
			rm -f md_file_toc generated_toc; \
		done | grep . && echo "Run './tools/mdtoc.sh <md-file>' to update the 'Table of Contents' in the Markdown files reported above." && exit 1 || \
		echo "$@: OK"

.PHONY: check_doc_links
check_doc_links: ## Check Markdown files for valid links
	@pip3 show requests > /dev/null || pip install requests
	@python3 tools/python/verify_doc_links.py
	@echo "$@: OK"

.PHONY: verify
verify: check_license check_mdtoc check_doc_links lint unit_test report ## Run all verification targets: check_license, check_mdtoc, lint, unit_test, report
	@echo "$@: OK"

.PHONY: distribution
distribution: venv ## Create a distribution and upload to test.PyPi.org
	@echo "NOTE: Using test.PyPi.org -- edit Makefile to target real PyPi index"
	@twine --version > /dev/null 2>&1 || pip install twine
	@cd sdk/python && \
		rm -rf dist/ && \
		python3 setup.py sdist && \
		twine check dist/* && \
		twine upload --repository testpypi dist/*

.PHONY: build
build: ## Create GO vendor directories with all dependencies
	go mod vendor
	# Extract go licenses into a single file. This assume licext is install globally through
	# npm install -g license-extractor
	# See https://github.com/arei/license-extractor
	licext --mode merge --source vendor/ --target third_party/license.txt --overwrite
	# Delete vendor directory
	rm -rf vendor

.PHONY: build-release-template
build-release-template: ## Build KFP Tekton release deployment templates
	@mkdir -p install/$(KFP_TEKTON_RELEASE)
	@kustomize build manifests/kustomize/env/kfp-template -o install/$(KFP_TEKTON_RELEASE)/kfp-tekton.yaml

.PHONY: build-backend
build-backend: build-apiserver build-agent build-workflow build-cacheserver ## Verify apiserver, agent, and workflow build
	@echo "$@: OK"

.PHONY: build-apiserver
build-apiserver: ## Build apiserver
	go build -o apiserver ./backend/src/apiserver

.PHONY: build-agent
build-agent: ## Build agent
	go build -o agent ./backend/src/agent/persistence

.PHONY: build-workflow
build-workflow: ## Build workflow
	go build -o workflow ./backend/src/crd/controller/scheduledworkflow/*.go

.PHONY: build-cacheserver
build-cacheserver: ## Build cache
	go build -o cache ./backend/src/cache/*.go

.PHONY: build-backend-images
build-backend-images: \
	build-api-server-image \
	build-persistenceagent-image \
	build-metadata-writer-image \
	build-scheduledworkflow-image \
	build-cacheserver-image \
	## Build backend docker images
	@echo "$@: OK"

.PHONY: build-api-server-image
build-api-server-image: ## Build api-server docker image
	docker build -t ${DOCKER_REGISTRY}/api-server -f backend/Dockerfile .

.PHONY: build-persistenceagent-image
build-persistenceagent-image: ## Build persistenceagent docker image
	docker build -t ${DOCKER_REGISTRY}/persistenceagent -f backend/Dockerfile.persistenceagent .

.PHONY: build-metadata-writer-image
build-metadata-writer-image: ## Build metadata-writer docker image
	docker build -t ${DOCKER_REGISTRY}/metadata-writer -f backend/metadata_writer/Dockerfile .

.PHONY: build-scheduledworkflow-image
build-scheduledworkflow-image: ## Build scheduledworkflow docker image
	docker build -t ${DOCKER_REGISTRY}/scheduledworkflow -f backend/Dockerfile.scheduledworkflow .

.PHONY: build-cacheserver-image
build-cacheserver-image: ## Build cacheserver docker image
	docker build -t ${DOCKER_REGISTRY}/cache-server -f backend/Dockerfile.cacheserver .

.PHONY: run-go-unittests
run-go-unittests: \
	run-apiserver-unittests \
	run-common-unittests \
	run-crd-unittests \
	run-persistenceagent-unittests \
	run-cacheserver-unittests \
	## Verify go backend unit tests
	@echo "$@: OK"

run-apiserver-unittests: # apiserver golang unit tests
	go test -v -cover ./backend/src/apiserver/...

run-common-unittests: # common golang unit tests
	go test -v -cover ./backend/src/common/...

run-crd-unittests: # crd golang unit tests
	go test -v -cover ./backend/src/crd/...

run-persistenceagent-unittests: # persistence agent golang unit tests
	go test -v -cover ./backend/src/agent/...

run-cacheserver-unittests: # cache golang unit tests
	go test -v -cover ./backend/src/cache/...
