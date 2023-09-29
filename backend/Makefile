BUILD=build
MOD_ROOT=..
CSV_PATH=backend/third_party_licenses

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: license_apiserver license_persistence_agent license_cache_server license_swf license_viewer tekton_licenses

.PHONY: tekton_licenses
tekton_licenses: license_tekton_driver license_tekton_ehc license_tekton_ehw license_tekton_ktc license_tekton_ktw

.PHONY: clean
clean:
	rm -rf $(BUILD)

$(BUILD)/apiserver:
	GO111MODULE=on go build -o $(BUILD)/apiserver github.com/kubeflow/pipelines/backend/src/apiserver
$(BUILD)/persistence_agent:
	GO111MODULE=on go build -o $(BUILD)/persistence_agent github.com/kubeflow/pipelines/backend/src/agent/persistence
$(BUILD)/cache_server:
	GO111MODULE=on go build -o $(BUILD)/cache_server github.com/kubeflow/pipelines/backend/src/cache
$(BUILD)/swf:
	GO111MODULE=on go build -o $(BUILD)/swf github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow
$(BUILD)/viewer:
	GO111MODULE=on go build -o $(BUILD)/viewer github.com/kubeflow/pipelines/backend/src/crd/controller/viewer
$(BUILD)/license_tekton_driver:
	GO111MODULE=on go build -o $(BUILD)/license_tekton_driver github.com/kubeflow/pipelines/backend/src/v2/cmd/controller
$(BUILD)/license_tekton_ehc:
	GO111MODULE=on go build -o $(BUILD)/license_tekton_ehc github.com/kubeflow/pipelines/backend/src/v2/cmd/tekton-exithandler/controller
$(BUILD)/license_tekton_ehw:
	GO111MODULE=on go build -o $(BUILD)/license_tekton_ehw github.com/kubeflow/pipelines/backend/src/v2/cmd/tekton-exithandler/webhook
$(BUILD)/license_tekton_ktc:
	GO111MODULE=on go build -o $(BUILD)/license_tekton_ktc github.com/kubeflow/pipelines/backend/src/v2/cmd/tekton-kfptask/controller
$(BUILD)/license_tekton_ktw:
	GO111MODULE=on go build -o $(BUILD)/license_tekton_ktw github.com/kubeflow/pipelines/backend/src/v2/cmd/tekton-kfptask/webhook

# Update licenses info after dependencies changed.
# See README.md#updating-licenses-info section for more details.
.PHONY: license_apiserver
license_apiserver: $(BUILD)/apiserver
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/apiserver > $(CSV_PATH)/apiserver.csv
.PHONY: license_persistence_agent
license_persistence_agent: $(BUILD)/persistence_agent
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/agent/persistence > $(CSV_PATH)/persistence_agent.csv
.PHONY: license_cache_server
license_cache_server: $(BUILD)/cache_server
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/cache > $(CSV_PATH)/cache_server.csv
.PHONY: license_swf
license_swf: $(BUILD)/swf
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/crd/controller/scheduledworkflow > $(CSV_PATH)/swf.csv
.PHONY: license_viewer
license_viewer: $(BUILD)/viewer
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/crd/controller/viewer > $(CSV_PATH)/viewer.csv
.PHONY: license_tekton_driver
license_tekton_driver: $(BUILD)/license_tekton_driver
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/controller > $(CSV_PATH)/tekton-driver.csv
.PHONY: license_tekton_ehc
license_tekton_ehc: $(BUILD)/license_tekton_ehc
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/tekton-exithandler/controller > $(CSV_PATH)/tekton-exithandler-controller.csv
.PHONY: license_tekton_ehw
license_tekton_ehw: $(BUILD)/license_tekton_ehw
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/tekton-exithandler/webhook > $(CSV_PATH)/tekton-exithandler-webhook.csv
.PHONY: license_tekton_ktc
license_tekton_ktc: $(BUILD)/license_tekton_ktc
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/tekton-kfptask/controller > $(CSV_PATH)/tekton-kfptask-controller.csv
.PHONY: license_tekton_ktw
license_tekton_ktw: $(BUILD)/license_tekton_ktw
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/tekton-kfptask/webhook > $(CSV_PATH)/tekton-kfptask-webhook.csv

.PHONY: image_all
image_all: image_apiserver image_persistence_agent image_cache image_swf image_viewer image_visualization

.PHONY: image_apiserver
image_apiserver:
	cd $(MOD_ROOT) && docker build -t apiserver -f backend/Dockerfile .
.PHONY: image_persistence_agent
image_persistence_agent:
	cd $(MOD_ROOT) && docker build -t persistence-agent -f backend/Dockerfile.persistenceagent .
.PHONY: image_cache
image_cache:
	cd $(MOD_ROOT) && docker build -t cache-server -f backend/Dockerfile.cacheserver .
.PHONY: image_swf
image_swf:
	cd $(MOD_ROOT) && docker build -t scheduledworkflow -f backend/Dockerfile.scheduledworkflow .
.PHONY: image_viewer
image_viewer:
	cd $(MOD_ROOT) && docker build -t viewercontroller -f backend/Dockerfile.viewercontroller .
.PHONY: image_visualization
image_visualization:
	cd $(MOD_ROOT) && docker build -t visualization -f backend/Dockerfile.visualization .
