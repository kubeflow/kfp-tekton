// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"

	"github.com/ghodss/yaml"

	tektonV1Beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	tektonVersion     = "tekton.dev/v1beta1"
	tektonK8sResource = "PipelineRun"
)

func GetParameters(template []byte) (string, error) {
	return GetTektonParameters(template)
}

func GetTektonParameters(template []byte) (string, error) {
	wf, err := ValidatePipelineRun(template)
	if err != nil {
		return "", Wrap(err, "Failed to get parameters from the pipelineRun")
	}
	if wf.Spec.Params == nil {
		return "[]", nil
	}
	paramBytes, err := json.Marshal(wf.Spec.Params)

	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

func ValidatePipelineRun(template []byte) (*tektonV1Beta1.PipelineRun, error) {
	var wf tektonV1Beta1.PipelineRun
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to parse the parameter.")
	}
	if wf.APIVersion != tektonVersion {
		return nil, NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", tektonVersion, wf.APIVersion)
	}
	if wf.Kind != tektonK8sResource {
		return nil, NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", tektonK8sResource, wf.Kind)
	}
	return &wf, nil
}
