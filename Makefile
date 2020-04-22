.PHONY: help

#
# Configuration variables
#
VENVDIR?=.venv
VENV = $(VENVDIR)/bin

PYTHON3OK :=$(shell python3 --version 2>&1)
ifeq ($(PYTHON3OK),)
  brew -y install python3.6 python3-pip
endif

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: venv
venv: ## create virtual environment
		test -d $(VENV) || python3 -m venv $(VENVDIR)

.PHONY: activate
activate: venv	## activate venv
		. $(VENV)/activate && exec $(notdir $(SHELL))

.PHONY: show-venv
show-venv: ## show venv
		@$(VENV)/python -c "import sys; print('Python ' + sys.version.replace('\n',''))"
		@$(VENV)/pip --version
		@echo venv: $(VENVDIR)

.PHONY: clean-venv
clean-venv:	venv ## clean venv
		[ ! -d $(VENVDIR) ] || rm -rf $(VENVDIR)

.PHONY: install
install: ## Install sdk/python
	pip install -e sdk/python

.PHONY: test
test: ## Run sdk/python unit test
	sdk/python/tests/run_tests.sh

.PHONY: report
report: ## Run sdk/python kfp sample testing report
	sdk/python/tests/test_kfp_samples.sh

.PHONY: venv-activate
venv-activate: venv activate ## create venv, activate venv
