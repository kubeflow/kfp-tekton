# Copyright 2023 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BUILD=build
CSV_PATH=third_party_licenses
REGISTER=
IMG_TAG=

.PHONY: licenses
licenses: license_controller license_webhook

.PHONY: license_controller
license_controller: third_party_licenses
	go-licenses csv ./cmd/controller > $(CSV_PATH)/controller.csv

.PHONY: license_webhook
license_webhook: third_party_licenses
	go-licenses csv ./cmd/webhook > $(CSV_PATH)/webhook.csv

.PHONY: images
images: image_controller image_webhook

.PHONY: image_controller
image_controller:
	docker build -t $(REGISTER)controller$(IMG_TAG) -f Dockerfile.controller .

.PHONY: image_webhook
image_webhook:
	docker build -t $(REGISTER)webhook$(IMG_TAG) -f Dockerfile.webhook .

.PHONY: install_go_license
install_go_license:
	go install github.com/google/go-licenses@d483853

third_party_licenses:
	mkdir third_party_licenses

$(BUILD):
	mkdir -p $(BUILD)