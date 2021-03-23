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
	"fmt"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/util"
	"github.com/pkg/errors"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	// Dag Inputs
	DagParamContextName = paramPrefixKfpInternal + "context-name"

	ParamDagTaskSpec   = DriverParamTaskSpec
	ParamExecutionName = "kfp-execution-name"

	ExecutorDriverServiceAccount = "kfp-tekton"
)

// Const can not be refered via string pointers, so we use var here.
var (
	argoVariablePodName = "{{pod.name}}"
	pipelineCounter     = 0
)

type DagArgs struct {
	IRComponentMap             map[string]*pb.ComponentSpec
	IRTasks                    *[]*pb.PipelineTaskSpec
	DeploymentConfig           *pb.PipelineDeploymentConfig
	TaskToTaskTemplate         map[string]string
	DagDriverTemplateName      string
	RootDagDriverTaskName      string
	ExecutorDriverTemplateName string
	ExecutorTemplateName       string
	UID                        string
	Labels                     map[string]string
	PipelineName               string
}

type taskData struct {
	task *pb.PipelineTaskSpec
	// we may need more stuff put here
}

func Dag(args *DagArgs) (*workflowapi.PipelineRun, error) {
	// convenient local variables
	tasks := args.IRTasks
	deploymentConfig := args.DeploymentConfig
	executors := deploymentConfig.GetExecutors()
	var serviceAccountNames []workflowapi.PipelineRunSpecServiceAccountName

	// fill up basic data and dag-driver
	root := &workflowapi.PipelineRun{}
	root.APIVersion = "tekton.dev/v1beta1"
	root.Kind = "PipelineRun"

	root.Name = "pipelinerun-" + strconv.Itoa(pipelineCounter) + "-" + args.UID
	root.Labels = args.Labels
	pipelineCounter++

	defaultTaskSpec := `{"taskInfo":{"name":"hello-world-dag"},"inputs":{"parameters":{"text":{"runtimeValue":{"constantValue":{"stringValue":"Hello, World!"}}}}}}`
	root.Spec.Params = []workflowapi.Param{
		{Name: ParamDagTaskSpec,
			Value: workflowapi.ArrayOrString{
				Type:      workflowapi.ParamTypeString,
				StringVal: defaultTaskSpec,
			},
		},
	}

	dag := workflowapi.PipelineSpec{}
	root.Spec.PipelineSpec = &dag

	dag.Params = []workflowapi.ParamSpec{{Name: ParamDagTaskSpec, Type: "string"}}

	// Always need a dag-driver to prepare the context for child tasks
	dag.Tasks = []workflowapi.PipelineTask{
		{Name: args.RootDagDriverTaskName, TaskRef: &workflowapi.TaskRef{Name: args.DagDriverTemplateName},
			Params: []workflowapi.Param{
				{Name: ParamExecutionName,
					Value: workflowapi.ArrayOrString{Type: "string", StringVal: "root-dag-driver-$(context.pipelineRun.uid)"},
				},
				{Name: ParamDagTaskSpec,
					Value: workflowapi.ArrayOrString{Type: "string", StringVal: fmt.Sprintf("$(params.%s)", ParamDagTaskSpec)},
				},
			},
		},
	}

	// pipelineTasks := dag.Tasks
	taskMap := make(map[string]*taskData)
	for index, task := range *tasks {
		name := task.GetTaskInfo().GetName()
		if name == "" {
			return nil, errors.Errorf("Task name is empty for task with index %v and spec: %s", index, task.String())
		}
		sanitizedName := util.SanitizeK8sName(name)
		if taskMap[sanitizedName] != nil {
			return nil, errors.Errorf("Two tasks '%s' and '%s' in the DAG has the same sanitized name: %s", taskMap[sanitizedName].task.GetTaskInfo().GetName(), name, sanitizedName)
		}
		taskMap[sanitizedName] = &taskData{
			task: task,
		}
	}

	// generate tasks
	for idx, task := range *tasks {
		// TODO(yhwang): need to check if this is a DAG task
		// TODO(Bobgy): Move executor template generation out as a separate file.
		executorLabel := task.GetExecutorLabel()
		executorSpec := executors[executorLabel]
		if executorSpec == nil {
			return nil, errors.Errorf("Executor with label '%v' cannot be found in deployment config", executorLabel)
		}
		var executor workflowapi.PipelineTask
		executor.Name = util.SanitizeK8sName(executorLabel)

		argoTaskName := util.SanitizeK8sName(task.GetTaskInfo().GetName())
		marshaler := &jsonpb.Marshaler{}
		taskSpecInJson, err := marshaler.MarshalToString(task)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to marshal task spec to JSON: %s", task.String())
		}
		executorSpecInJson, err := marshaler.MarshalToString(executorSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to marshal executor spec to JSON: %s", executorSpec.String())
		}
		// TODO(Bobgy): Task outputs spec is deprecated. Get outputs spec from component output spec once data is ready.
		outputsSpec := task.GetOutputs()
		if outputsSpec == nil {
			// For tasks without outputs spec, marshal an emtpy outputs spec.
			outputsSpec = &pb.TaskOutputsSpec{}
		}
		// outputsSpecInJson, err := marshaler.MarshalToString(outputsSpec)
		// if err != nil {
		// 	return nil, errors.Wrapf(err, "Failed to marshal outputs spec to JSON: %s", task.GetOutputs().String())
		// }
		// parentContextNameValue := "{{inputs.parameters." + DagParamContextName + "}}"

		dependencies, err := getTaskDependencies(task.GetInputs())

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get task dependencies for task: %s", task.String())
		}

		// Convert dependency names to sanitized ones and check validity.
		for index, dependency := range *dependencies {
			sanitizedDependencyName := util.SanitizeK8sName(dependency)
			upstreamTask := taskMap[sanitizedDependencyName]
			if upstreamTask == nil {
				return nil, errors.Wrapf(err, "Failed to find dependency '%s' for task: %s", dependency, task.String())
			}
			upstreamTaskName := upstreamTask.task.GetTaskInfo().GetName()
			if upstreamTaskName != dependency {
				return nil, errors.Wrapf(err, "Found slightly different dependency task name '%s', expecting '%s' for task: %s", upstreamTaskName, dependency, task.String())
			}
			(*dependencies)[index] = sanitizedDependencyName
		}

		// first task need to wait for the DAG driver
		if idx == 0 {
			*dependencies = append(*dependencies, args.RootDagDriverTaskName)
		}

		dag.Tasks = append(
			dag.Tasks,
			workflowapi.PipelineTask{
				Name:     argoTaskName + "-executor-driver",
				TaskRef:  &workflowapi.TaskRef{Name: args.ExecutorDriverTemplateName},
				RunAfter: *dependencies,
				Params: []workflowapi.Param{
					{Name: ParamExecutionName, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: argoTaskName + "-executor-driver-$(context.pipelineRun.uid)"}},
					{Name: DriverParamParentContextName, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: "$(tasks." + args.RootDagDriverTaskName + ".results.context-name)"}},
					{Name: ExecutorParamTemplateNameSpace, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: "$(context.pipelineRun.namespace)"}},
					{Name: ExecutorParamTemplateName, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: args.TaskToTaskTemplate[task.GetTaskInfo().GetName()]}},
					{Name: ExecutorParamTaskSpec, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: taskSpecInJson}},
					{Name: ExecutorParamExecutorSpec, Value: workflowapi.ArrayOrString{Type: workflowapi.ParamTypeString, StringVal: executorSpecInJson}},
				},
			},
			workflowapi.PipelineTask{
				Name:     argoTaskName,
				TaskRef:  &workflowapi.TaskRef{Name: args.TaskToTaskTemplate[task.TaskInfo.Name]},
				RunAfter: []string{argoTaskName + "-executor-driver"},
			},
		)

		serviceAccountNames = append(serviceAccountNames,
			workflowapi.PipelineRunSpecServiceAccountName{TaskName: argoTaskName + "-executor-driver",
				ServiceAccountName: ExecutorDriverServiceAccount,
			})
	}

	root.Spec.ServiceAccountNames = serviceAccountNames

	return root, nil
}

func getTaskDependencies(inputsSpec *pb.TaskInputsSpec) (*[]string, error) {
	dependencies := make(map[string]bool)
	for _, parameter := range inputsSpec.GetParameters() {
		if parameter.GetTaskOutputParameter() != nil {
			producerTask := parameter.GetTaskOutputParameter().GetProducerTask()
			if producerTask == "" {
				return nil, errors.Errorf("Invalid task input parameter spec, producer task is empty: %v", parameter.String())
			}
			dependencies[producerTask] = true
		}
	}
	dependencyList := make([]string, 0, len(dependencies)+1)
	for dependency := range dependencies {
		dependencyList = append(dependencyList, dependency)
	}
	return &dependencyList, nil
}

// TODO(Bobgy): figure out a better way to generate unique names
var globalDagCount = 0

func getUniqueDagName() string {
	globalDagCount = globalDagCount + 1
	return fmt.Sprintf("dag-%x", globalDagCount)
}
