package compiler

import (
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8score "k8s.io/api/core/v1"
)

func (c *pipelineCompiler) Importer(
	name string, parentDAG string,
	task *pipelinespec.PipelineTaskSpec,
	component *pipelinespec.ComponentSpec,
	importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec,
) error {
	if component == nil {
		return fmt.Errorf("Importer: component spec must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return fmt.Errorf("Importer: marshaling component spec to proto JSON failed: %w", err)
	}
	importerJson, err := marshaler.MarshalToString(importer)
	if err != nil {
		return fmt.Errorf("Importer: marlshaling importer spec to proto JSON failed: %w", err)
	}
	taskJson, err := marshaler.MarshalToString(task)
	if err != nil {
		return fmt.Errorf("Importer: marshaling task spec to proto JSON failed: %w", err)
	}

	launcherArgs := []string{
		"--executor_type", "importer",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--importer_spec", inputValue(paramImporter),
		"--pipeline_name", c.spec.PipelineInfo.GetName(),
		"--run_id", runID(),
		"--pod_name",
		"$(KFP_POD_NAME)",
		"--pod_uid",
		"$(KFP_POD_UID)",
		"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
		"$(METADATA_GRPC_SERVICE_HOST)",
		"--mlmd_server_port",
		"$(METADATA_GRPC_SERVICE_PORT)",
	}
	mlmdConfigOptional := true

	pipelineTask := &pipeline.PipelineTask{
		Name: name,
		TaskSpec: &pipeline.EmbeddedTask{
			Metadata: pipeline.PipelineTaskMetadata{
				Annotations: map[string]string{
					"pipelines.kubeflow.org/v2_pipeline": "true",
				},
				Labels: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
					"pipeline-uid":                        c.uid,
				},
			},
			TaskSpec: pipeline.TaskSpec{
				Params: []pipeline.ParamSpec{
					{Name: paramTask, Type: "string"},
					{Name: paramComponent, Type: "string"},
					{Name: paramImporter, Type: "string"},
				},
				Results: []pipeline.TaskResult{
					{Name: paramExecutionID, Description: "execution id"},
					{Name: paramExecutorInput, Description: "executor input"},
				},
				Steps: []pipeline.Step{
					{Container: k8score.Container{
						Name:    "importer-main",
						Image:   c.launcherImage,
						Command: []string{"launcher-v2"},
						Args:    launcherArgs,
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
						}},
					}},
				},
			},
		},
		RunAfter: append(task.GetDependentTasks(), getDAGDriverTaskName(parentDAG)),
		Params: []pipeline.Param{
			{
				Name:  paramTask,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: taskJson},
			},
			{
				Name:  paramComponent,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: componentJson},
			},
			{
				Name:  paramImporter,
				Value: pipeline.ArrayOrString{Type: "string", StringVal: importerJson},
			},
		},
	}
	c.addPipelineTask(pipelineTask)
	return err
}
