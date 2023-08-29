// Copyright 2018 The Kubeflow Authors
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
	"context"
	"encoding/json"

	"sigs.k8s.io/yaml"

	tektonV1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	workflowapiV1beta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	tektonBetaGroup   = "tekton.dev/v1beta1"
	tektonVersion     = "tekton.dev/v1"
	tektonK8sResource = "PipelineRun"
)

func UnmarshalParameters(paramsString string) ([]tektonV1.Param, error) {
	if paramsString == "" {
		return nil, nil
	}
	var params []tektonV1.Param
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, NewInternalServerError(err, "Parameters have wrong format")
	}
	return params, nil
}

func MarshalParameters(params []tektonV1.Param) (string, error) {
	if params == nil {
		return "[]", nil
	}
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

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

func ValidatePipelineRun(template []byte) (*Workflow, error) {
	var wf tektonV1.PipelineRun
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to parse the parameter.")
	}
	if wf.APIVersion == tektonBetaGroup {
		var prV1beta1 workflowapiV1beta.PipelineRun
		err := yaml.Unmarshal(template, &prV1beta1)
		if err != nil {
			return nil, NewInvalidInputErrorWithDetails(err, "Failed to parse the V1beta1 PipelineRun template.")
		}
		ctx := context.Background()
		prV1beta1.ConvertTo(ctx, &wf)
		wf.APIVersion = tektonVersion
	}
	if wf.APIVersion != tektonVersion {
		return nil, NewInvalidInputError("Unsupported tekton version. Expected: %v. Received: %v", tektonVersion, wf.APIVersion)
	}
	if wf.Kind != tektonK8sResource {
		return nil, NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", tektonK8sResource, wf.Kind)
	}
	return NewWorkflow(&wf), nil
}
