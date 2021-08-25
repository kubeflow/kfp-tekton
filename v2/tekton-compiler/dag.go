package compiler

import (
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *pipelineCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) error {
	if name != "root" {
		return fmt.Errorf("SubDAG not implemented yet")
	}
	err := addImplicitDependencies(dagSpec)
	if err != nil {
		return err
	}

	task := &pipelinespec.PipelineTaskSpec{}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if name == "root" {
		// runtime config is input to the entire pipeline (root DAG)
		runtimeConfig = c.job.GetRuntimeConfig()
	}
	driverTask, outputs, err := c.dagDriverTask(name+"-dag", componentSpec, task, runtimeConfig)
	if err != nil {
		return err
	}

	marshaler := jsonpb.Marshaler{}

	//push current dag outputs to the dagOutputs queue
	c.dagPipelineTasks = append(c.dagPipelineTasks, driverTask)
	c.addPipelineTask(driverTask)

	// create PipelineTask (taskRef: system-container-driver-<uuid>) for each task in a dag
	for _, kfpTask := range dagSpec.GetTasks() {
		taskJson, err := marshaler.MarshalToString(kfpTask)
		if err != nil {
			return fmt.Errorf("DAG: marshaling task spec to proto JSON failed: %w", err)
		}
		componentJson, err := marshaler.MarshalToString(c.spec.Components[kfpTask.ComponentRef.Name])
		if err != nil {
			return fmt.Errorf("DAG: marshaling component spec to proto JSON failed: %w", err)
		}
		c.addPipelineTask(&pipeline.PipelineTask{
			Name:     kfpTask.GetTaskInfo().GetName() + "-driver",
			TaskRef:  &pipeline.TaskRef{Name: c.containerTask.Name},
			RunAfter: append(kfpTask.GetDependentTasks(), driverTask.Name),
			Params: []pipeline.Param{
				{
					Name:  paramDAGContextID,
					Value: pipeline.ArrayOrString{Type: "string", StringVal: outputs.contextID},
				},
				{
					Name:  paramDAGExecutionID,
					Value: pipeline.ArrayOrString{Type: "string", StringVal: outputs.executionID},
				},
				{
					Name:  paramTask,
					Value: pipeline.ArrayOrString{Type: "string", StringVal: taskJson},
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
	}

	return err
}

type dagDriverOutputs struct {
	contextID, executionID string
}

func (c *pipelineCompiler) dagDriverTask(
	name string,
	component *pipelinespec.ComponentSpec,
	task *pipelinespec.PipelineTaskSpec,
	runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig,
) (*pipeline.PipelineTask, *dagDriverOutputs, error) {
	if component == nil {
		return nil, nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return nil, nil, fmt.Errorf("dagDriverTask: marlshaling component spec to proto JSON failed: %w", err)
	}
	runtimeConfigJson := "{}"
	if runtimeConfig != nil {
		runtimeConfigJson, err = marshaler.MarshalToString(runtimeConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
	}

	// generate DAG task template if needed
	dagTaskName := c.addDAGDriverTaskTemplate()

	t := &pipeline.PipelineTask{
		Name:    name,
		TaskRef: &pipeline.TaskRef{Name: dagTaskName},
		Params: []pipeline.Param{
			{
				Name:  paramComponent,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: componentJson},
			},
			{
				Name:  paramRuntimeConfig,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: runtimeConfigJson},
			},
			{
				Name:  paramRunId,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: runID()},
			},
		},
	}
	return t, &dagDriverOutputs{
		contextID:   taskOutputParameter(name, paramContextID),
		executionID: taskOutputParameter(name, paramExecutionID),
	}, nil
}

//Generate task template for DAG driver
func (c *pipelineCompiler) addDAGDriverTaskTemplate() string {
	if c.dagTask != nil {
		return c.dagTask.Name
	}

	driver := &pipeline.Task{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: fmt.Sprintf("system-dag-driver-%s", c.uid),
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
				{Name: paramComponent, Type: "string"},     // --component
				{Name: paramRuntimeConfig, Type: "string"}, // --runtime_config
				{Name: paramParentContextID, Type: "string",
					Default: &pipeline.ArrayOrString{Type: "string", StringVal: ""}},
				{Name: paramRunId, Type: "string"},
			},
			Results: []pipeline.TaskResult{
				{Name: paramExecutionID, Description: "execution id"},
				{Name: paramContextID, Description: "context id"},
			},
			Steps: []pipeline.Step{
				{Container: k8score.Container{
					Name:    "dag-driver-main",
					Image:   c.driverImage,
					Command: []string{"driver"},
					Args: []string{
						"--type", "ROOT_DAG",
						"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
						"--run_id", inputParameter(paramRunId),
						"--component", inputParameter(paramComponent),
						"--runtime_config", inputParameter(paramRuntimeConfig),
						"--execution_id_path", outputPath(paramExecutionID),
						"--context_id_path", outputPath(paramContextID),
						"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
						"$(METADATA_GRPC_SERVICE_HOST)",
						"--mlmd_server_port",
						"$(METADATA_GRPC_SERVICE_PORT)",
					},
					Env: []k8score.EnvVar{{
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
	c.addTask(driver, driver.Name)
	c.dagTask = driver
	return driver.Name
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
		// TODO(Bobgy): add implicit dependencies introduced by artifacts
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
