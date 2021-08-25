package compiler

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/protobuf/proto"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Options struct {
	// optional, use official image if not provided
	LauncherImage string
	// optional
	DriverImage string
	// optional
	PipelineRoot string
	// TODO(Bobgy): add an option -- dev mode, ImagePullPolicy should only be Always in dev mode.
}

type PipelineCRDSet struct {
	PipelineRun *pipeline.PipelineRun
	Tasks       []*pipeline.Task
}

func Compile(jobArg *pipelinespec.PipelineJob, opts *Options) (*PipelineCRDSet, error) {
	spec, err := getPipelineSpec(jobArg)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}

	// uid
	uid := uuid.New().String()

	//initialization
	pipelineRun := &pipeline.PipelineRun{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: spec.GetPipelineInfo().GetName(),
			Annotations: map[string]string{
				"sidecar.istio.io/inject":            "false",
				"pipelines.kubeflow.org/v2_pipeline": "true",
			},
			Labels: map[string]string{
				"pipelines.kubeflow.org/v2_component": "true",
				"pipeline-uid":                        uid,
			},
		},
		Spec: pipeline.PipelineRunSpec{
			PipelineSpec:       &pipeline.PipelineSpec{},
			ServiceAccountName: "pipeline-runner",
		},
	}

	jobMsg := proto.Clone(jobArg)
	job, ok := jobMsg.(*pipelinespec.PipelineJob)
	if !ok {
		return nil, fmt.Errorf("bug: cloned pipeline job message does not have expected type")
	}

	compiler := &pipelineCompiler{
		pipelineRun:      pipelineRun,
		tasks:            make(map[string]*pipeline.Task),
		driverImage:      "gcr.io/ml-pipeline/kfp-driver:latest",
		launcherImage:    "gcr.io/ml-pipeline/kfp-launcher-v2:latest",
		job:              job,
		spec:             spec,
		uid:              uid,
		dagPipelineTasks: make([]*pipeline.PipelineTask, 5),
	}
	if opts != nil {
		if opts.DriverImage != "" {
			compiler.driverImage = opts.DriverImage
		}
		if opts.LauncherImage != "" {
			compiler.launcherImage = opts.LauncherImage
		}
		if opts.PipelineRoot != "" {
			if job.RuntimeConfig == nil {
				job.RuntimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
			}
			job.RuntimeConfig.GcsOutputDirectory = opts.PipelineRoot
		}
	}

	// compile
	Accept(job, compiler)

	tasks := make([]*pipeline.Task, 0, 5)
	for _, task := range compiler.tasks {
		tasks = append(tasks, task)
	}
	return &PipelineCRDSet{PipelineRun: compiler.pipelineRun, Tasks: tasks}, nil
}

type pipelineCompiler struct {
	// inputs
	job  *pipelinespec.PipelineJob
	spec *pipelinespec.PipelineSpec
	// state
	pipelineRun      *pipeline.PipelineRun
	tasks            map[string]*pipeline.Task
	driverImage      string
	launcherImage    string
	uid              string
	dagPipelineTasks []*pipeline.PipelineTask
	dagTask          *pipeline.Task
	containerTask    *pipeline.Task
}

func (c *pipelineCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return fmt.Errorf("importer not implemented yet")
}
func (c *pipelineCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return fmt.Errorf("resolver not implemented yet")
}

var errAlreadyExists = fmt.Errorf("template already exists")

func (c *pipelineCompiler) addTask(t *pipeline.Task, name string) (string, error) {
	t.Name = c.taskName(name)
	_, ok := c.tasks[t.Name]
	if ok {
		return "", fmt.Errorf("template name=%q: %w", t.Name, errAlreadyExists)
	}

	c.tasks[t.Name] = t
	return t.Name, nil
}

func (c *pipelineCompiler) addPipelineTask(t *pipeline.PipelineTask) {
	c.pipelineRun.Spec.PipelineSpec.Tasks = append(c.pipelineRun.Spec.PipelineSpec.Tasks, *t)
}

func (c *pipelineCompiler) taskName(componentName string) string {
	// TODO(Bobgy): sanitize component name, because argo template names
	// must be valid Kubernetes resource names.
	return componentName
}

const (
	paramComponent       = "component"      // component spec
	paramTask            = "task"           // task spec
	paramRuntimeConfig   = "runtime-config" // job runtime config, pipeline level inputs
	paramDAGContextID    = "dag-context-id"
	paramParentContextID = "parent-context-id"
	paramDAGExecutionID  = "dag-execution-id"
	paramExecutionID     = "execution-id"
	paramContextID       = "context-id"
	paramExecutorInput   = "executor-input"
	paramRunId           = "run-id"
)

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "$(context.pipelineRun.uid)"
}

func workflowParameter(name string) string {
	return fmt.Sprintf("{{workflow.parameters.%s}}", name)
}

// In a container template, refer to inputs to the template.
func inputValue(parameter string) string {
	return fmt.Sprintf("$(params.%s)", parameter)
}

// In a DAG/steps task, refer to inputs to task params.
func inputParameter(parameter string) string {
	return fmt.Sprintf("$(params.%s)", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("$(results.%s.path)", parameter)
}

func taskOutputParameter(task string, param string) string {
	//tasks.<taskName>.results.<resultName>
	return fmt.Sprintf("$(tasks.%s.results.%s)", task, param)
}
