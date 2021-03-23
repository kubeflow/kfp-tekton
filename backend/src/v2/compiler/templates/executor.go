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
	apiv1 "k8s.io/api/core/v1"
)

// TODO(Bobgy): make image configurable
const (
	entrypointImage     = "yihongwang/kfp-entrypoint-dev"
	entrypointImageRef  = "0.8"
	entrypointImageFull = entrypointImage + ":" + entrypointImageRef
)

const (
	TemplateNameExecutor = "kfp-executor"
	templateNameDummy    = "kfp-dummy"

	// Executor inputs
	ExecutorParamTaskSpec          = DriverParamTaskSpec
	ExecutorParamExecutorSpec      = DriverParamExecutorSpec
	ExecutorParamContextName       = paramPrefixKfpInternal + "context-name"
	ExecutorParamTemplateName      = paramPrefixKfpInternal + "executor-template-name"
	ExecutorParamTemplateNameSpace = paramPrefixKfpInternal + "executor-template-namespace"
	ExecutorParamOutputsSpec       = PublisherParamOutputsSpec

	executorInternalParamPodName       = paramPrefixKfpInternal + "pod-name"
	executorInternalParamPodSpecPatch  = paramPrefixKfpInternal + "pod-spec-patch"
	executorInternalArtifactParameters = paramPrefixKfpInternal + "parameters"
)

// Executor generate task template for executor
func Executor() *workflowapi.Task {
	executor := workflowapi.Task{}
	var spec workflowapi.TaskSpec

	executor.APIVersion = "tekton.dev/v1beta1"
	executor.Kind = "Task"

	spec.StepTemplate = &apiv1.Container{
		VolumeMounts: []apiv1.VolumeMount{
			{MountPath: "/kfp/entrypoint", Name: "data"},
		},
	}
	spec.Volumes = []apiv1.Volume{
		{Name: "data", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}},
	}

	spec.Steps = []workflowapi.Step{
		{Container: apiv1.Container{
			Name:    "entrypoint-init",
			Image:   entrypointImageFull,
			Command: []string{"cp"},
			Args:    []string{"/bin/kfp-entrypoint", "/kfp/entrypoint/entrypoint"},
		}},
		{Container: apiv1.Container{
			Name:    "task-executor",
			Image:   "dummy",
			Command: []string{"/bin/sh", "-c"},
		}},
	}
	executor.Spec = spec
	return &executor
}
