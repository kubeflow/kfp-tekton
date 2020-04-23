# Copyright 2020 kubeflow.org
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

#
# Configuration variables
#
VENV ?= .venv
export VIRTUAL_ENV := $(abspath ${VENV})
export PATH := ${VIRTUAL_ENV}/bin:${PATH}

# Use the command from https://stackoverflow.com/a/35730328/5601796
.PHONY: help 
help: ## Display the Make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: venv
venv: $(VENV)/bin/activate ## Create and activate virtual environment
$(VENV)/bin/activate: sdk/python/setup.py
	@test -d $(VENV) || python3 -m venv $(VENV)
	pip install -e sdk/python
	@touch $(VENV)/bin/activate

.PHONY: test
test: venv ## Run kfp/tekton unit test
	@sdk/python/tests/run_tests.sh

.PHONY: report
report: ## Report kfp sample stats
	@cd sdk/python/tests && ./test_kfp_samples.sh
