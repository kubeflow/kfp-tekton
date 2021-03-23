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

package main

import (
	"strconv"

	"github.com/google/uuid"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/templates"
	"github.com/pkg/errors"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	rootDagDriverTaskName = "driver-kfp-root"
	rootDagTaskName       = "kfp-dag-root-main"
)

const (
	templateNameExecutorDriver    = "kfp-executor-driver"
	templateNameDagDriver         = "kfp-dag-driver"
	templateNameExecutorPublisher = "kfp-executor-publisher"
	rootDagTaskSpec               = "root-dag-task-spec"
	paramExecutionName            = "kfp-execution-name"
)

// PipelineCRDSet contains PipelineRuns and Tasks
type PipelineCRDSet struct {
	PipelineRuns *[]*workflowapi.PipelineRun
	Tasks        *[]*workflowapi.Task
}

// CompilePipelineSpec convert PipelineSpec to CRDs
func CompilePipelineSpec(
	pipelineSpec *pb.PipelineSpec,
	deploymentConfig *pb.PipelineDeploymentConfig,
) (*PipelineCRDSet, error) {

	// validation
	if pipelineSpec.GetPipelineInfo().GetName() == "" {
		return nil, errors.New("Name is empty")
	}

	rev, err := generateCRDs(pipelineSpec, deploymentConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to generate workflow spec")
	}

	return rev, nil
}

func generateCRDs(
	pipelineSpec *pb.PipelineSpec,
	deploymentConfig *pb.PipelineDeploymentConfig,
) (*PipelineCRDSet, error) {

	var revTasks []*workflowapi.Task
	var revPipelineRuns []*workflowapi.PipelineRun
	var uID = uuid.New().String()
	var executorTaskMap = make(map[string]string)
	var pipelineName = pipelineSpec.GetPipelineInfo().GetName()
	var labels = map[string]string{
		"pipeline-uid": uID,
	}

	tasks := pipelineSpec.GetTasks()
	// replace the tasks list with the tasks in a DAG
	if pipelineSpec.Root.GetDag() != nil {
		var dagTasks []*pb.PipelineTaskSpec
		for _, task := range pipelineSpec.Root.GetDag().GetTasks() {
			dagTasks = append(dagTasks, task)
		}
		tasks = dagTasks
	}

	// generate helper templates
	// dag driver and executor driver are fixed task template, should be
	// able to directly create them during installation
	executorDriver := templates.Driver(false)
	executorDriver.Name = templateNameExecutorDriver
	dagDriver := templates.Driver(true)
	dagDriver.Name = templateNameDagDriver

	revTasks = append(revTasks, dagDriver, executorDriver)

	// generate executor tasks' CRDs
	for idx, task := range tasks {
		executorTemplate := templates.Executor()
		executorTemplate.Name = "task-" + strconv.Itoa(idx) + "-" + uID
		executorTemplate.Labels = labels
		executorTaskMap[task.TaskInfo.Name] = executorTemplate.Name
		revTasks = append(revTasks, executorTemplate)
	}

	// executorPublisher := templates.Publisher(common.PublisherType_EXECUTOR)
	// executorPublisher.Name = templateNameExecutorPublisher
	// executorTemplates := templates.Executor(templateNameExecutorDriver, templateNameExecutorPublisher)

	// root.PipelineSpec = rootDag
	// TODO: make a generic default value

	root, err := templates.Dag(&templates.DagArgs{
		IRComponentMap:             pipelineSpec.Components,
		IRTasks:                    &tasks,
		DeploymentConfig:           deploymentConfig,
		TaskToTaskTemplate:         executorTaskMap,
		RootDagDriverTaskName:      rootDagTaskName,
		DagDriverTemplateName:      templateNameDagDriver,
		ExecutorDriverTemplateName: templateNameExecutorDriver,
		ExecutorTemplateName:       templates.TemplateNameExecutor,
		UID:                        uID,
		Labels:                     labels,
		PipelineName:               pipelineName,
	})
	if err != nil {
		return nil, err
	}

	revPipelineRuns = append(revPipelineRuns, root)

	//parentContextName := "{{tasks." + rootDagDriverTaskName + ".outputs.parameters." + templates.DriverParamContextName + "}}"
	//root.PipelineSpec.Tasks = append(root.PipelineSpec.Tasks, subDag.Tasks...)

	// []workflowapi.Template{root, *subDag, *executorDriver, *dagDriver, *executorPublisher}
	// for _, template := range executorTemplates {
	// 	spec.PipelineSpec.Tasks = append(spec.PipelineSpec.Tasks, *template)
	// }

	return &PipelineCRDSet{&revPipelineRuns, &revTasks}, nil
}
