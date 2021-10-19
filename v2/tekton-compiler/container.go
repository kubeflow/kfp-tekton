package compiler

import (
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	volumePathKFPLauncher = "/tekton/home"
	volumeNameKFPLauncher = "kfp-launcher"
)

func (c *pipelineCompiler) Container(
	name string, nameInDAG string, parentDAG string,
	task *pipelinespec.PipelineTaskSpec,
	component *pipelinespec.ComponentSpec,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
) error {

	if component == nil {
		return fmt.Errorf("workflowCompiler.Container: component spec must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling component spec to proto JSON failed: %w", err)
	}
	containerJson, err := marshaler.MarshalToString(container)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling pipeline container spec to proto JSON failed: %w", err)
	}
	dagDriverName := getDAGDriverTaskName(parentDAG)
	containerDriverName := getContainerDriverTaskName(nameInDAG)
	// Need dag driver's output for container driver
	dagDriverOutput := dagDriverOutputs{
		contextID:   taskOutputParameter(dagDriverName, paramContextID),
		executionID: taskOutputParameter(dagDriverName, paramExecutionID),
	}
	// create a container driver here
	taskJson, err := marshaler.MarshalToString(task)
	if err != nil {
		return fmt.Errorf("Container: marshaling task spec to proto JSON failed: %w", err)
	}

	c.addPipelineTask(&pipeline.PipelineTask{
		Name: containerDriverName,
		TaskRef: &pipeline.TaskRef{
			APIVersion: "kfp-driver.tekton.dev/v1alpha1",
			Kind:       "KFPDriver",
			Name:       "kfp-driver",
		},
		RunAfter: append(task.GetDependentTasks(), dagDriverName),
		Params: []pipeline.Param{
			{
				Name:  paramType,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: "CONTAINER"},
			},
			{
				Name:  paramPipelineName,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			{
				Name:  paramRunId,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: runID()},
			},
			{
				Name:  paramDAGContextID,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: dagDriverOutput.contextID},
			},
			{
				Name:  paramDAGExecutionID,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: dagDriverOutput.executionID},
			},
			{
				Name:  paramComponent,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: componentJson},
			},
			{
				Name:  paramTask,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: taskJson},
			},
			{
				Name:  paramContainer,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: containerJson},
			},
			{
				Name:  paramExecutionID,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: paramExecutionID},
			},
			{
				Name:  paramContextID,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: paramContextID},
			},
			{
				Name:  paramExecutorInput,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: paramExecutorInput},
			},
			{
				Name:  paramCachedDecision,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: paramCachedDecision},
			},
		},
	})

	// need container driver's output for executor
	containerDriverOutputs := containerDriverOutputs{
		executorInput: taskOutputParameter(containerDriverName, paramExecutorInput),
		executionID:   taskOutputParameter(containerDriverName, paramExecutionID),
		cached:        taskOutputParameter(containerDriverName, paramCachedDecision),
	}
	t := c.containerExecutorTemplate(name, container, c.spec.PipelineInfo.GetName())
	// TODO(Bobgy): how can we avoid template name collisions?
	// For Tekton we use a Task to represent the executor driver
	c.addTask(t, name+"-"+c.uid)
	if err != nil {
		return err
	}
	// create a PipelineTask to refer to container task here
	// create container driver in dag while iterating through the tasks
	// name collisions?
	c.addPipelineTask(&pipeline.PipelineTask{
		Name:     nameInDAG,
		TaskRef:  &pipeline.TaskRef{Name: t.Name},
		RunAfter: []string{containerDriverName},
		WhenExpressions: pipeline.WhenExpressions{
			{
				Input:    containerDriverOutputs.cached,
				Operator: "in",
				Values:   []string{"false"},
			},
		},
		Params: []pipeline.Param{
			{
				Name:  paramExecutorInput,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: containerDriverOutputs.executorInput},
			},
			{
				Name:  paramExecutionID,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: containerDriverOutputs.executionID},
			},
			{
				Name:  paramComponent,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: componentJson},
			},
			{
				Name:  paramRunId,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: runID()},
			},
		},
	})
	return err
}

type containerDriverOutputs struct {
	executorInput string
	executionID   string
	cached        string
}

// func (c *pipelineCompiler) containerDriverTask(name string) *containerDriverOutputs {
// 	// c.addContainerDriverTaskTemplate()
// 	outputs := &containerDriverOutputs{
// 		executorInput: taskOutputParameter(name, paramExecutorInput),
// 		executionID:   taskOutputParameter(name, paramExecutionID),
// 		cached:        taskOutputParameter(name, paramCachedDecision),
// 	}
// 	return outputs
// }

// func (c *pipelineCompiler) addContainerDriverTaskTemplate() string {
// 	if c.containerTask != nil {
// 		return c.containerTask.Name
// 	}

// 	driver := &pipeline.Task{
// 		TypeMeta: k8smeta.TypeMeta{
// 			APIVersion: "tekton.dev/v1beta1",
// 			Kind:       "Task",
// 		},
// 		ObjectMeta: k8smeta.ObjectMeta{
// 			Name: fmt.Sprintf("system-container-driver-%s", c.uid),
// 			Annotations: map[string]string{
// 				"pipelines.kubeflow.org/v2_pipeline": "true",
// 			},
// 			Labels: map[string]string{
// 				"pipelines.kubeflow.org/v2_component": "true",
// 				"pipeline-uid":                        c.uid,
// 			},
// 		},
// 		Spec: pipeline.TaskSpec{
// 			Params: []pipeline.ParamSpec{
// 				{Name: paramComponent, Type: "string"},      // --component
// 				{Name: paramTask, Type: "string"},           // --task
// 				{Name: paramDAGContextID, Type: "string"},   // --dag-context-id
// 				{Name: paramDAGExecutionID, Type: "string"}, // --dag-execution-id
// 				{Name: paramRunId, Type: "string"},
// 			},
// 			Results: []pipeline.TaskResult{
// 				{Name: paramExecutionID, Description: "execution id"},
// 				{Name: paramExecutorInput, Description: "executor input"},
// 			},
// 			Steps: []pipeline.Step{
// 				{Container: k8score.Container{
// 					Name:    "container-driver-main",
// 					Image:   c.driverImage,
// 					Command: []string{"driver"},
// 					Args: []string{
// 						"--type", "CONTAINER",
// 						"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
// 						"--run_id", inputValue(paramRunId),
// 						"--dag_context_id", inputValue(paramDAGContextID),
// 						"--dag_execution_id", inputValue(paramDAGExecutionID),
// 						"--component", inputValue(paramComponent),
// 						"--task", inputValue(paramTask),
// 						"--execution_id_path", outputPath(paramExecutionID),
// 						"--executor_input_path", outputPath(paramExecutorInput),
// 						"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
// 						"$(METADATA_GRPC_SERVICE_HOST)",
// 						"--mlmd_server_port",
// 						"$(METADATA_GRPC_SERVICE_PORT)",
// 					},
// 					Env: []k8score.EnvVar{{
// 						Name:  "METADATA_GRPC_SERVICE_HOST",
// 						Value: "metadata-grpc-service.kubeflow.svc.cluster.local",
// 					}, {
// 						Name:  "METADATA_GRPC_SERVICE_PORT",
// 						Value: "8080",
// 					}},
// 				}},
// 			},
// 		},
// 	}

// 	c.addTask(driver, driver.Name)
// 	c.containerTask = driver
// 	return driver.Name
// }

func (c *pipelineCompiler) containerExecutorTemplate(
	name string, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	pipelineName string,
) *pipeline.Task {
	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	launcherCmd := []string{
		volumePathKFPLauncher + "/launch",
		"--pipeline_name", pipelineName,
		"--run_id", inputValue(paramRunId),
		"--execution_id", inputValue(paramExecutionID),
		"--executor_input", inputValue(paramExecutorInput),
		"--component_spec", inputValue(paramComponent),
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
	return &pipeline.Task{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, c.uid),
			Annotations: map[string]string{
				"pipelines.kubeflow.org/v2_pipeline": "true",
			},
			Labels: map[string]string{
				"pipelines.kubeflow.org/v2_component": "true",
				"pipeline-uid":                        c.uid,
			},
		},
		Spec: pipeline.TaskSpec{
			Params: []pipeline.ParamSpec{
				{Name: paramExecutorInput, Type: "string"}, // --executor-input
				{Name: paramExecutionID, Type: "string"},   // --execution-id
				{Name: paramComponent, Type: "string"},     // --component
				{Name: paramRunId, Type: "string"},         // --run-id
			},
			Results: []pipeline.TaskResult{
				{Name: paramExecutionID, Description: "execution id"},
				{Name: paramExecutorInput, Description: "executor input"},
			},
			Steps: []pipeline.Step{
				// step 1: copy launcher
				{Container: k8score.Container{
					Name:            "kfp-launcher",
					Image:           c.launcherImage,
					Command:         []string{"launcher-v2", "--copy", "/tekton/home/launch"},
					ImagePullPolicy: "Always",
				}},
				// wrap user program with executor
				{Container: k8score.Container{
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
				}},
			},
		},
	}
}
