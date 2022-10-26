// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tektoncompiler

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8score "k8s.io/api/core/v1"
)

const (
	volumeNameKFPLauncher = "kfp-launcher"
	kfpLauncherPath       = "/tekton/home/launch"
)

func (c *pipelinerunCompiler) Container(taskName, compRef string,
	task *pipelinespec.PipelineTaskSpec,
	component *pipelinespec.ComponentSpec,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
) error {

	err := c.saveComponentSpec(compRef, component)
	if err != nil {
		return err
	}
	err = c.saveComponentImpl(compRef, container)
	if err != nil {
		return err
	}

	componentSpec, err := c.useComponentSpec(compRef)
	if err != nil {
		return fmt.Errorf("component spec for %q not found", compRef)
	}
	taskSpecJson, err := stablyMarshalJSON(task)
	if err != nil {
		return err
	}
	containerImpl, err := c.useComponentImpl(compRef)
	if err != nil {
		return err
	}

	return c.containerDriverTask(taskName, &containerDriverInputs{
		component:    componentSpec,
		task:         taskSpecJson,
		container:    containerImpl,
		parentDag:    getDAGDriverTaskName(c.CurrentDag()),
		taskDef:      task,
		containerDef: container,
	})
}

type containerDriverOutputs struct {
	// podSpecPatch string
	// break down podSpecPath to the following
	executionId    string
	executiorInput string
	cached         string
	condition      string
}

type containerDriverInputs struct {
	component      string
	task           string
	taskDef        *pipelinespec.PipelineTaskSpec
	container      string
	containerDef   *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	parentDag      string
	iterationIndex string // optional, when this is an iteration task
}

func (c *pipelinerunCompiler) containerDriverTask(name string, inputs *containerDriverInputs) error {

	containerDriverName := getContainerDriverTaskName(name)
	// task driver
	c.addPipelineTask(&pipelineapi.PipelineTask{
		Name: containerDriverName,
		TaskRef: &pipelineapi.TaskRef{
			APIVersion: "kfp-driver.tekton.dev/v1alpha1",
			Kind:       "KFPDriver",
			Name:       "kfp-driver",
		},
		RunAfter: append(inputs.taskDef.GetDependentTasks(), inputs.parentDag),
		Params: []pipelineapi.Param{
			// "--type", "CONTAINER",
			{
				Name:  paramNameType,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: "CONTAINER"},
			},
			// "--pipeline_name", c.spec.GetPipelineInfo().GetName(),
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run_id", runID(),
			{
				Name:  paramNameRunId,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: c.uid},
			},
			// "--dag_execution_id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: taskOutputParameter(inputs.parentDag, paramExecutionID)},
			},
			// "--component"
			{
				Name:  paramComponent,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.component},
			},
			// "--task"
			{
				Name:  paramTask,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.task},
			},
			// "--container"
			{
				Name:  paramContainer,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.container},
			},
			// "--iteration_index", inputValue(paramIterationIndex),
			{
				Name:  paramNameIterationIndex,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.iterationIndex},
			},
			// output params below
			// // "--execution_id",
			// {
			// 	Name:  paramNameExecutionId,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramExecutionID},
			// },
			// // "--executor_input",
			// {
			// 	Name:  paramNameExecutorInput,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramExecutorInput},
			// },
			// // "--function_to_execute"
			// {
			// 	Name:  paramNameFunctionToExecute,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramFunctionToExecute},
			// },
			// // "--cached_decision", outputPath(paramCachedDecision),
			// {
			// 	Name:  paramNameCachedDecision,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramCachedDecision},
			// },
			// // "--condition", outputPath(paramCondition),
			// {
			// 	Name:  paramNameCondition,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramCondition},
			// },
		},
	})

	// need container driver's output for executor
	containerDriverOutputs := containerDriverOutputs{
		executionId:    taskOutputParameter(containerDriverName, paramExecutionID),
		condition:      taskOutputParameter(containerDriverName, paramCondition),
		executiorInput: taskOutputParameter(containerDriverName, paramExecutorInput),
		cached:         taskOutputParameter(containerDriverName, paramCachedDecision),
	}

	t := c.containerExecutorTemplate(name, inputs.containerDef, c.spec.PipelineInfo.GetName())
	c.addPipelineTask(&pipelineapi.PipelineTask{
		Name:     name,
		TaskSpec: t,
		RunAfter: []string{containerDriverName},
		WhenExpressions: pipelineapi.WhenExpressions{
			{
				Input:    containerDriverOutputs.cached,
				Operator: "in",
				Values:   []string{"false"},
			},
		},
		Params: []pipelineapi.Param{
			{
				Name:  paramExecutorInput,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: containerDriverOutputs.executiorInput},
			},
			{
				Name:  paramExecutionID,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: containerDriverOutputs.executionId},
			},
			{
				Name:  paramRunId,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: c.uid},
			},
			{
				Name:  paramComponentSpec,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.component},
			},
		},
	})

	return nil
}

func (c *pipelinerunCompiler) containerExecutorTemplate(
	name string, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	pipelineName string,
) *pipelineapi.EmbeddedTask {
	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	// userCmdArgs = append(userCmdArgs, "--executor_input", "{{$}}", "--function_to_execute", inputValue(paramFunctionToExecute))
	launcherCmd := []string{
		kfpLauncherPath,
		"--pipeline_name", pipelineName,
		"--run_id", inputValue(paramRunId),
		"--execution_id", inputValue(paramExecutionID),
		"--executor_input", inputValue(paramExecutorInput),
		"--component_spec", inputValue(paramComponentSpec),
		"--pod_name",
		"$(KFP_POD_NAME)",
		"--pod_uid",
		"$(KFP_POD_UID)",
		"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
		"$(METADATA_GRPC_SERVICE_HOST)",
		"--mlmd_server_port",
		"$(METADATA_GRPC_SERVICE_PORT)",
		"--", // separater before user command and args
	}
	mlmdConfigOptional := true
	return &pipelineapi.EmbeddedTask{
		Metadata: pipelineapi.PipelineTaskMetadata{
			Annotations: map[string]string{
				"pipelines.kubeflow.org/v2_pipeline": "true",
			},
			Labels: map[string]string{
				"pipelines.kubeflow.org/v2_component": "true",
				"pipeline-uid":                        c.uid,
			},
		},
		TaskSpec: pipelineapi.TaskSpec{
			Params: []pipelineapi.ParamSpec{
				{Name: paramExecutorInput, Type: "string"}, // --executor_input
				{Name: paramExecutionID, Type: "string"},   // --execution_id
				{Name: paramRunId, Type: "string"},         // --run_id
				{Name: paramComponentSpec, Type: "string"}, // --component_spec
			},
			Results: []pipelineapi.TaskResult{
				{Name: paramExecutionID, Description: "execution id"},
				{Name: paramExecutorInput, Description: "executor input"},
			},
			Steps: []pipelineapi.Step{
				// step 1: copy launcher
				{
					Name:            "kfp-launcher",
					Image:           c.launcherImage,
					Command:         []string{"launcher-v2", "--copy", kfpLauncherPath},
					ImagePullPolicy: "Always",
				},
				// wrap user program with executor
				{
					Name:    "user-main",
					Image:   container.Image,
					Command: launcherCmd,
					Args:    userCmdArgs,
					EnvFrom: []k8score.EnvFromSource{{
						ConfigMapRef: &k8score.ConfigMapEnvSource{
							LocalObjectReference: k8score.LocalObjectReference{
								Name: "metadata-grpc-configmap",
							},
							Optional: &mlmdConfigOptional,
						},
					}},
					Env: []k8score.EnvVar{{
						Name: "KFP_POD_NAME",
						ValueFrom: &k8score.EnvVarSource{
							FieldRef: &k8score.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					}, {
						Name: "KFP_POD_UID",
						ValueFrom: &k8score.EnvVarSource{
							FieldRef: &k8score.ObjectFieldSelector{
								FieldPath: "metadata.uid",
							},
						},
					}, {
						Name:  "METADATA_GRPC_SERVICE_HOST",
						Value: "metadata-grpc-service.kubeflow.svc.cluster.local",
					}, {
						Name:  "METADATA_GRPC_SERVICE_PORT",
						Value: "8080",
					}},
				},
			},
		},
	}
}
