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
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func (c *pipelinerunCompiler) DAG(taskName, compRef string, task *pipelinespec.PipelineTaskSpec, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("compiling DAG %q: %w", taskName, err)
		}
	}()

	err = c.saveComponentSpec(compRef, componentSpec)
	if err != nil {
		return err
	}

	if err := c.addDagTask(taskName, compRef, task); err != nil {
		return err
	}
	return nil
}

func (c *pipelinerunCompiler) addDagTask(name, compRef string, task *pipelinespec.PipelineTaskSpec) error {
	driverTaskName := getDAGDriverTaskName(name)
	componentSpecStr, err := c.useComponentSpec(compRef)
	if err != nil {
		return err
	}

	inputs := dagDriverInputs{
		component: componentSpecStr,
	}

	if name == compiler.RootComponentName {
		// runtime config is input to the entire pipeline (root DAG)
		inputs.runtimeConfig = c.job.GetRuntimeConfig()
	} else {
		//sub-dag and task shall not be nil
		if task == nil {
			return fmt.Errorf("invalid sub-dag")
		}
		taskSpecJson, err := stablyMarshalJSON(task)
		if err != nil {
			return err
		}
		task.DependentTasks = append(task.DependentTasks, getDAGDriverTaskName(c.CurrentDag()))
		inputs.task = taskSpecJson
		inputs.deps = task.GetDependentTasks()
		inputs.parentDagID = c.CurrentDag()
	}

	driver, err := c.dagDriverTask(driverTaskName, &inputs)
	if err != nil {
		return err
	}
	c.addPipelineTask(driver)
	return nil
}

func (c *pipelinerunCompiler) dagDriverTask(
	name string,
	inputs *dagDriverInputs,
) (*pipelineapi.PipelineTask, error) {
	if inputs == nil || len(inputs.component) == 0 {
		return nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	runtimeConfigJson := "{}"
	if inputs.runtimeConfig != nil {
		rtStr, err := stablyMarshalJSON(inputs.runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
		runtimeConfigJson = rtStr
	}

	t := &pipelineapi.PipelineTask{
		Name: name,
		TaskRef: &pipelineapi.TaskRef{
			APIVersion: "kfp-driver.tekton.dev/v1alpha1",
			Kind:       "KFPDriver",
			Name:       "kfp-driver",
		},
		Params: []pipelineapi.Param{
			// "--type"
			{
				Name:  paramNameType,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.getDagType()},
			},
			// "--pipeline_name"
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run_id"
			{
				Name:  paramNameRunId,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: c.uid},
			},
			// "--dag_execution_id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.getParentDagID()},
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
			// "--runtime_config"
			{
				Name:  paramNameRuntimeConfig,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: runtimeConfigJson},
			},
			// "--iteration_index"
			{
				Name:  paramNameIterationIndex,
				Value: pipelineapi.ArrayOrString{Type: "string", StringVal: inputs.getIterationIndex()},
			},
			//output below, make them constances for now
			// "--execution_id", outputPath(paramExecutionID), change from execution_id_path to execution_id
			// {
			// 	Name:  paramNameExecutionId,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramExecutionID},
			// },
			// // "--iteration_count", outputPath(paramIterationCount),
			// {
			// 	Name:  paramNameIterationCount,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramIterationCount},
			// },
			// // "--condition_path", outputPath(paramCondition),
			// {
			// 	Name:  paramNameCondition,
			// 	Value: pipelineapi.ArrayOrString{Type: "string", StringVal: paramCondition},
			// },
		},
	}
	if len(inputs.deps) > 0 {
		t.RunAfter = inputs.deps
	}
	return t, nil
}

type dagDriverInputs struct {
	parentDagID    string                                  // parent DAG execution ID. optional, the root DAG does not have parent
	component      string                                  // input placeholder for component spec
	task           string                                  // optional, the root DAG does not have task spec.
	runtimeConfig  *pipelinespec.PipelineJob_RuntimeConfig // optional, only root DAG needs this
	iterationIndex string                                  // optional, iterator passes iteration index to iteration tasks
	deps           []string
}

func (i *dagDriverInputs) getParentDagID() string {
	if i.parentDagID == "" {
		return "0"
	}
	return taskOutputParameter(getDAGDriverTaskName(i.parentDagID), paramExecutionID)
}

func (i *dagDriverInputs) getDagType() string {
	if i.runtimeConfig != nil {
		return "ROOT_DAG"
	}
	return "DAG"
}

func (i *dagDriverInputs) getIterationIndex() string {
	if i.iterationIndex == "" {
		return "-1"
	}
	return i.iterationIndex
}

func addImplicitDependencies(dagSpec *pipelinespec.DagSpec) error {
	for _, task := range dagSpec.GetTasks() {
		wrap := func(err error) error {
			return fmt.Errorf("failed to add implicit deps: %w", err)
		}
		addDep := func(producer string) error {
			if _, ok := dagSpec.GetTasks()[producer]; !ok {
				return fmt.Errorf("unknown producer task %q in DAG", producer)
			}
			if task.DependentTasks == nil {
				task.DependentTasks = make([]string, 0)
			}
			// add the dependency if it's not already added
			found := false
			for _, dep := range task.DependentTasks {
				if dep == producer {
					found = true
					break
				}
			}
			if !found {
				task.DependentTasks = append(task.DependentTasks, producer)
			}
			return nil
		}
		for _, input := range task.GetInputs().GetParameters() {
			switch input.GetKind().(type) {
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
				if err := addDep(input.GetTaskOutputParameter().GetProducerTask()); err != nil {
					return wrap(err)
				}
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
				return wrap(fmt.Errorf("task final status not supported yet"))
			default:
				// other parameter input types do not introduce implicit dependencies
			}
		}
		for _, input := range task.GetInputs().GetArtifacts() {
			switch input.GetKind().(type) {
			case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
				if err := addDep(input.GetTaskOutputArtifact().GetProducerTask()); err != nil {
					return wrap(err)
				}
			default:
				// other artifact input types do not introduce implicit dependencies
			}
		}
	}
	return nil
}

func getLeafNodes(dagSpec *pipelinespec.DagSpec) []string {
	leaves := make(map[string]int)
	tasks := dagSpec.GetTasks()
	alldeps := make([]string, 0)
	for _, task := range tasks {
		leaves[task.GetTaskInfo().GetName()] = 0
		alldeps = append(alldeps, task.GetDependentTasks()...)
	}
	for _, dep := range alldeps {
		delete(leaves, dep)
	}
	rev := make([]string, 0, len(leaves))
	for dep := range leaves {
		rev = append(rev, dep)
	}
	return rev
}

// depends builds an enhanced depends string for argo.
// Argo DAG normal dependencies run even when upstream tasks are skipped, which
// is not what we want. Using enhanced depends, we can be strict that upstream
// tasks must be succeeded.
// https://argoproj.github.io/argo-workflows/enhanced-depends-logic/
func depends(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, dep := range deps {
		if index > 0 {
			builder.WriteString(" && ")
		}
		builder.WriteString(dep)
		builder.WriteString(".Succeeded")
	}
	return builder.String()
}

// Exit handler task happens no matter the state of the upstream tasks
func depends_exit_handler(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, dep := range deps {
		if index > 0 {
			builder.WriteString(" || ")
		}
		for inner_index, task_status := range []string{".Succeeded", ".Skipped", ".Failed", ".Errored"} {
			if inner_index > 0 {
				builder.WriteString(" || ")
			}
			builder.WriteString(dep)
			builder.WriteString(task_status)
		}
	}
	return builder.String()
}
