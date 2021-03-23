// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

import (
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
)

// TODO(Bobgy): make image configurable
// gcr.io/gongyuan-pipeline-test/kfp-driver:latest
const (
	driverImage     = "docker.io/yihongwang/kfp-driver-dev"
	driverImageRef  = "0.8"
	driverImageFull = driverImage + ":" + driverImageRef
)

const (
	paramPrefixKfpInternal = "kfp-"
	outputPathPodSpecPatch = "/kfp/outputs/pod-spec-patch.json"
)

const (
	// Inputs
	DriverParamParentContextName = paramPrefixKfpInternal + "parent-context-name"
	DriverParamExecutionName     = paramPrefixKfpInternal + "execution-name"
	DriverParamDriverType        = paramPrefixKfpInternal + "driver-type"
	DriverParamTaskSpec          = paramPrefixKfpInternal + "task-spec"
	DriverParamExecutorSpec      = paramPrefixKfpInternal + "executor-spec"
	// Outputs
	DriverParamExecutionId  = paramPrefixKfpInternal + "execution-id"
	DriverParamContextName  = paramPrefixKfpInternal + "context-name"
	DriverParamPodSpecPatch = paramPrefixKfpInternal + "pod-spec-patch"
)

// Do not modify this, this should be constant too.
// This needs to be used in string pointers, so it is "var" instead of "const".
var (
	mlmdExecutionName = "kfp-executor-{{pod.name}}"
)

// Driver get Task template
func Driver(isDag bool) *workflowapi.Task {
	driver := &workflowapi.Task{}
	driver.APIVersion = "tekton.dev/v1beta1"
	driver.Kind = "Task"

	var spec workflowapi.TaskSpec

	params := []workflowapi.ParamSpec{
		{Name: "kfp-task-spec", Type: "string"},
		{Name: DriverParamParentContextName, Type: "string",
			Default: &workflowapi.ArrayOrString{Type: "string", StringVal: ""}},
		{Name: "kfp-execution-name", Type: "string"},
	}

	results := []workflowapi.TaskResult{
		{Name: "execution-id", Description: "execution id"},
	}

	spec.Steps = []workflowapi.Step{
		{Container: k8sv1.Container{Image: driverImageFull,
			Command: []string{"/bin/kfp-driver"}}},
	}

	if isDag {
		spec.Steps[0].Container.Name = "dag-driver-main"
		spec.Steps[0].Container.Args = []string{
			"--logtostderr",
			"--driver_type=DAG",
			"--mlmd_url=metadata-grpc-service.kubeflow.svc.cluster.local:8080",
			"--parent_context_name=$(params.kfp-parent-context-name)",
			"--execution_name=kfp-$(params.kfp-execution-name)",
			"--output_path_execution_id=$(results.execution-id.path)",
			"--output_path_context_name=$(results.context-name.path)",
			"--task_spec=$(params.kfp-task-spec)",
		}

		results = append(results,
			workflowapi.TaskResult{Name: "context-name", Description: "context name"},
		)
	} else {
		spec.Steps[0].Container.Name = "executor-driver-main"
		spec.Steps[0].Container.Args = []string{
			"--logtostderr",
			"--driver_type=EXECUTOR",
			"--mlmd_url=metadata-grpc-service.kubeflow.svc.cluster.local:8080",
			"--parent_context_name=$(params.kfp-parent-context-name)",
			"--execution_name=kfp-$(params.kfp-execution-name)",
			"--output_path_execution_id=$(results.execution-id.path)",
			"--task_spec=$(params.kfp-task-spec)",
			"--executor_spec=$(params.kfp-executor-spec)",
			"--output_path_pod_spec_patch=$(results.pod-spec-patch.path)",
			"--task_template_namespace=$(params.kfp-executor-template-namespace)",
			"--task_template_name=$(params.kfp-executor-template-name)",
		}

		params = append(params,
			workflowapi.ParamSpec{Name: "kfp-executor-spec", Type: "string"},
			workflowapi.ParamSpec{Name: "kfp-executor-template-name", Type: "string"},
			workflowapi.ParamSpec{Name: "kfp-executor-template-namespace", Type: "string"},
		)

		results = append(results,
			workflowapi.TaskResult{Name: "pod-spec-patch", Description: "pod spec patch"},
		)
	}

	spec.Params = params
	spec.Results = results
	driver.Spec = spec

	return driver
}
