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
	"sort"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
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

func Compile(jobArg *pipelinespec.PipelineJob, opts *Options) (*pipelineapi.PipelineRun, error) {
	// clone jobArg, because we don't want to change it
	jobMsg := proto.Clone(jobArg)
	job, ok := jobMsg.(*pipelinespec.PipelineJob)
	if !ok {
		return nil, fmt.Errorf("bug: cloned pipeline job message does not have expected type")
	}
	if job.RuntimeConfig == nil {
		job.RuntimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
	}
	if job.GetRuntimeConfig().GetParameterValues() == nil {
		job.RuntimeConfig.ParameterValues = map[string]*structpb.Value{}
	}
	spec, err := compiler.GetPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}
	deploy, err := compiler.GetDeploymentConfig(spec)
	if err != nil {
		return nil, err
	}
	// fill root component default paramters to PipelineJob
	specParams := spec.GetRoot().GetInputDefinitions().GetParameters()
	for name, param := range specParams {
		_, ok := job.RuntimeConfig.ParameterValues[name]
		if !ok && param.GetDefaultValue() != nil {
			job.RuntimeConfig.ParameterValues[name] = param.GetDefaultValue()
		}
	}

	// initialization
	pr := &pipelineapi.PipelineRun{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "PipelineRun",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName: retrieveLastValidString(spec.GetPipelineInfo().GetName()) + "-",
			Annotations: map[string]string{
				"pipelines.kubeflow.org/v2_pipeline":  "true",
				"tekton.dev/artifact_bucket":          "mlpipeline",
				"tekton.dev/artifact_endpoint":        "minio-service.kubeflow:9000",
				"tekton.dev/artifact_endpoint_scheme": "http://",
			},
			Labels: map[string]string{
				"pipelines.kubeflow.org/v2_component": "true",
			},
		},
		Spec: pipelineapi.PipelineRunSpec{
			PipelineSpec: &pipelineapi.PipelineSpec{},
		},
	}
	c := &pipelinerunCompiler{
		pr: pr,
		// TODO(chensun): release process and update the images.
		launcherImage: "gcr.io/ml-pipeline-test/dev/kfp-launcher-v2@sha256:4513cf5c10c252d94f383ce51a890514799c200795e3de5e90f91b98b2e2f959",
		job:           job,
		spec:          spec,
		dagStack:      make([]string, 0, 10),
		executors:     deploy.GetExecutors(),
	}
	if opts != nil {
		if opts.LauncherImage != "" {
			c.launcherImage = opts.LauncherImage
		}
		if opts.PipelineRoot != "" {
			job.RuntimeConfig.GcsOutputDirectory = opts.PipelineRoot
		}
	}

	// compile
	err = Accept(job, c)

	return c.pr, err
}

type TektonVisitor interface {
	// receive task and component reference and use these information to create
	// container driver and executor tasks
	Container(taskName, compRef string,
		task *pipelinespec.PipelineTaskSpec,
		component *pipelinespec.ComponentSpec,
		container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error

	// use task and component information to create importer task
	Importer(name string,
		task *pipelinespec.PipelineTaskSpec,
		component *pipelinespec.ComponentSpec,
		importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error

	// Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error

	// create root dag and sub-dag driver task
	DAG(taskName, compRef string,
		task *pipelinespec.PipelineTaskSpec, // could be sub-dag
		component *pipelinespec.ComponentSpec,
		dag *pipelinespec.DagSpec) error

	// put the current DAG into the stack. when processing tasks inside a DAG, this could be used
	// to know which DAG they belong to
	PushDagStack(dag string)

	// pop the DAG when finishing the processing
	PopDagStack() string

	// get current DAG when processing the tasks inside a DAG
	CurrentDag() string
}

type pipelinerunDFS struct {
	spec    *pipelinespec.PipelineSpec
	deploy  *pipelinespec.PipelineDeploymentConfig
	visitor TektonVisitor
	// Records which DAG components are visited, map key is component name.
	visited map[string]bool
}

func Accept(job *pipelinespec.PipelineJob, v TektonVisitor) error {
	if job == nil {
		return nil
	}
	// TODO(Bobgy): reserve root as a keyword that cannot be user component names
	spec, err := compiler.GetPipelineSpec(job)
	if err != nil {
		return err
	}
	deploy, err := compiler.GetDeploymentConfig(spec)
	if err != nil {
		return err
	}
	state := &pipelinerunDFS{
		spec:    spec,
		deploy:  deploy,
		visitor: v,
		visited: make(map[string]bool),
	}
	// start to traverse the DAG, starting from the root node
	return state.dfs(compiler.RootComponentName, compiler.RootComponentName, nil, spec.GetRoot())
}

// taskName:  the task's name in a DAG
// compRef:   the component name that this task refers to
// task:      the task's task spec
// component: the task's component spec
func (state *pipelinerunDFS) dfs(taskName, compRef string, task *pipelinespec.PipelineTaskSpec, component *pipelinespec.ComponentSpec) error {
	// each component is only visited once
	// TODO(Bobgy): return an error when circular reference detected
	if state.visited[taskName] {
		return nil
	}
	state.visited[taskName] = true
	if component == nil {
		return nil
	}
	if state == nil {
		return fmt.Errorf("dfs: unexpected value state=nil")
	}

	componentError := func(err error) error {
		return fmt.Errorf("error processing component name=%q: %w", compRef, err)
	}

	executorLabel := component.GetExecutorLabel()
	if executorLabel != "" {
		executor, ok := state.deploy.GetExecutors()[executorLabel]
		if !ok {
			return componentError(fmt.Errorf("executor(label=%q) not found in deployment config", executorLabel))
		}
		container := executor.GetContainer()
		if container != nil {
			return state.visitor.Container(taskName, compRef, task, component, container)
		}
		importer := executor.GetImporter()
		if importer != nil {
			return state.visitor.Importer(taskName, task, component, importer)
		}

		return componentError(fmt.Errorf("executor(label=%q): non-container and non-importer executor not implemented", executorLabel))
	}
	dag := component.GetDag()
	if dag == nil { // impl can only be executor or dag
		return componentError(fmt.Errorf("unknown component implementation: %s", component))
	}
	// move this from DAG() to here
	err := addImplicitDependencies(dag)
	if err != nil {
		return err
	}

	// from here, start to process DAG task, push self to DAG stack first
	state.visitor.PushDagStack(taskName)

	tasks := dag.GetTasks()
	keys := make([]string, 0, len(tasks))
	for key := range tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		task, ok := tasks[key]
		if !ok {
			return componentError(fmt.Errorf("this is a bug: cannot find key %q in tasks", key))
		}
		refName := task.GetComponentRef().GetName()
		if refName == "" {
			return componentError(fmt.Errorf("component ref name is empty for task name=%q", task.GetTaskInfo().GetName()))
		}
		subComponent, ok := state.spec.Components[refName]
		if !ok {
			return componentError(fmt.Errorf("cannot find component ref name=%q", refName))
		}

		// check the dependencies
		state.checkDependencies(task, dag)
		err := state.dfs(key, refName, task, subComponent)
		if err != nil {
			return err
		}
	}

	// pop the dag stack, assume no need to use the dag stack when processing DAG
	// for sub-dag, it can also get its parent dag
	state.visitor.PopDagStack()

	// TODO: revisit this
	// if name != "root" {
	// 	// non-root DAG also has dependencies
	// 	state.checkDependencies(task)
	// }

	// process tasks before DAG component, so that all sub-tasks are already
	// ready by the time the DAG component is visited.
	return state.visitor.DAG(taskName, compRef, task, component, dag)
}

func retrieveLastValidString(s string) string {
	sections := strings.Split(s, "/")
	return sections[len(sections)-1]
}

type pipelinerunCompiler struct {
	// inputs
	job       *pipelinespec.PipelineJob
	spec      *pipelinespec.PipelineSpec
	executors map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec
	// state
	pr             *pipelineapi.PipelineRun
	launcherImage  string
	dagStack       []string
	componentSpecs map[string]string
	containerSpecs map[string]string
}

// if the dependency is a component with DAG, then replace the dependency with DAG's leaf nodes
func (state *pipelinerunDFS) checkDependencies(task *pipelinespec.PipelineTaskSpec, dag *pipelinespec.DagSpec) {
	if task.GetTriggerPolicy() != nil && task.GetTriggerPolicy().GetStrategy().String() == "ALL_UPSTREAM_TASKS_COMPLETED" {
		// don't change the exit handler's deps, let it depend on the dag
		return
	}
	tasks := task.GetDependentTasks()
	newDeps := make([]string, 0)
	for _, depTask := range tasks {
		if taskSpec, ok := dag.GetTasks()[depTask]; ok {
			if comp, ok := state.spec.Components[taskSpec.GetComponentRef().GetName()]; ok {
				depDag := comp.GetDag()
				//depends on a DAG
				if depDag != nil {
					newDeps = append(newDeps, getLeafNodes(depDag, state.spec)...)
					continue
				} else {
					newDeps = append(newDeps, depTask)
				}
			}
		}
	}
	task.DependentTasks = newDeps
}

func getLeafNodes(dagSpec *pipelinespec.DagSpec, spec *pipelinespec.PipelineSpec) []string {
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
		refName := tasks[dep].GetComponentRef().GetName()
		if comp, ok := spec.Components[refName]; ok {
			if comp.GetDag() != nil {
				rev = append(rev, getLeafNodes(comp.GetDag(), spec)...)
			} else {
				rev = append(rev, dep)
			}
		}
	}
	return rev
}

func (c *pipelinerunCompiler) PushDagStack(dagName string) {
	c.dagStack = append(c.dagStack, dagName)
}

func (c *pipelinerunCompiler) PopDagStack() string {
	lsize := len(c.dagStack)
	if lsize > 0 {
		rev := c.dagStack[lsize-1]
		c.dagStack = c.dagStack[:lsize-1]
		return rev
	}
	return ""
}

func (c *pipelinerunCompiler) CurrentDag() string {
	lsize := len(c.dagStack)
	if lsize > 0 {
		return c.dagStack[lsize-1]
	}
	return ""
}

func (c *pipelinerunCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return fmt.Errorf("resolver not implemented yet")
}

// Add a PipelineTask into a Pipeline as one of the tasks in its PipelineSpec
func (c *pipelinerunCompiler) addPipelineTask(t *pipelineapi.PipelineTask) {
	if c.pr.Spec.PipelineSpec.Tasks == nil {
		c.pr.Spec.PipelineSpec.Tasks = make([]pipelineapi.PipelineTask, 0)
	}
	c.pr.Spec.PipelineSpec.Tasks = append(c.pr.Spec.PipelineSpec.Tasks, *t)
}

/* no use of finally at this moment, use dependency to fullfil the exit handler for now */
func (c *pipelinerunCompiler) addPipelineFinallyTask(t *pipelineapi.PipelineTask) {
	if c.pr.Spec.PipelineSpec.Finally == nil {
		c.pr.Spec.PipelineSpec.Finally = make([]pipelineapi.PipelineTask, 0)
	}
	c.pr.Spec.PipelineSpec.Finally = append(c.pr.Spec.PipelineSpec.Finally, *t)
}

// WIP: store component spec, task spec and executor spec in annotations

const (
	prefixComponents = "components-"
	prefixContainers = "implementations-"
)

func (c *pipelinerunCompiler) saveComponentSpec(name string, spec *pipelinespec.ComponentSpec) error {
	if c.componentSpecs == nil {
		c.componentSpecs = make(map[string]string)
	}
	return c.putValueToMap(prefixComponents+name, spec, c.componentSpecs)
}

func (c *pipelinerunCompiler) useComponentSpec(name string) (string, error) {
	return c.getValueFromMap(prefixComponents+name, c.componentSpecs)
}

func (c *pipelinerunCompiler) saveComponentImpl(name string, msg proto.Message) error {
	if c.containerSpecs == nil {
		c.containerSpecs = make(map[string]string)
	}
	return c.putValueToMap(prefixContainers+name, msg, c.containerSpecs)
}

func (c *pipelinerunCompiler) useComponentImpl(name string) (string, error) {
	return c.getValueFromMap(prefixContainers+name, c.containerSpecs)
}

func (c *pipelinerunCompiler) putValueToMap(name string, msg proto.Message, maps map[string]string) error {
	if _, alreadyExists := maps[name]; alreadyExists {
		return fmt.Errorf("componentSpec %q already exists", name)
	}
	json, err := stablyMarshalJSON(msg)
	if err != nil {
		return fmt.Errorf("saving component spec of %q to pipelinerunCompiler: %w", name, err)
	}
	maps[name] = json
	return nil
}

func (c *pipelinerunCompiler) getValueFromMap(name string, maps map[string]string) (string, error) {
	rev, exists := maps[name]
	if !exists {
		return "", fmt.Errorf("using component spec: failed to find componentSpec %q", name)
	}
	return rev, nil
}

const (
	paramComponent      = "component"      // component spec
	paramTask           = "task"           // task spec
	paramContainer      = "container"      // container spec
	paramImporter       = "importer"       // importer spec
	paramRuntimeConfig  = "runtime-config" // job runtime config, pipeline level inputs
	paramParentDagID    = "parent-dag-id"
	paramExecutionID    = "execution-id"
	paramIterationCount = "iteration-count"
	paramIterationIndex = "iteration-index"
	paramExecutorInput  = "executor-input"
	paramDriverType     = "driver-type"
	paramCachedDecision = "cached-decision" // indicate hit cache or not
	paramPodSpecPatch   = "pod-spec-patch"  // a strategic patch merged with the pod spec
	paramCondition      = "condition"       // condition = false -> skip the task
	paramRunId          = "run-id"
	paramComponentSpec  = "component-spec"

	paramNameType             = "type"
	paramNamePipelineName     = "pipeline_name"
	paramNameRunId            = "run_id"
	paramNameDagExecutionId   = "dag_execution_id"
	paramNameRuntimeConfig    = "runtime_config"
	paramNameIterationIndex   = "iteration_index"
	paramNameExecutionId      = "execution_id"
	paramNameIterationCount   = "iteration_count"
	paramNameCondition        = "condition"
	paramNameCachedDecision   = "cached_decision"
	paramNamePodSpecPatchPath = "pod_spec_patch_path"
	paramNameExecutorInput    = "executor_input"
)

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "$(context.pipelineRun.uid)"
}

// In a container template, refer to inputs to the template.
func inputValue(parameter string) string {
	return fmt.Sprintf("$(params.%s)", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("$(results.%s.path)", parameter)
}

func taskOutputParameter(task string, param string) string {
	//tasks.<taskName>.results.<resultName>
	return fmt.Sprintf("$(tasks.%s.results.%s)", task, param)
}

func getDAGDriverTaskName(dagName string) string {
	if dagName == compiler.RootComponentName {
		// root dag
		return fmt.Sprintf("%s-system-dag-driver", dagName)
	}
	// sub dag
	return fmt.Sprintf("%s-dag-driver", dagName)
}

func getDAGPubTaskName(dagName string) string {
	if dagName == compiler.RootComponentName {
		// root dag
		return fmt.Sprintf("%s-system-dag-pub-driver", dagName)
	}
	// sub dag
	return fmt.Sprintf("%s-dag-pub-driver", dagName)
}

func getContainerDriverTaskName(name string) string {
	return fmt.Sprintf("%s-driver", name)
}

// Usually drivers should take very minimal amount of CPU and memory, but we
// set a larger limit for extreme cases.
// Note, these are empirical data.
// No need to make this configurable, because we will instead call drivers using argo HTTP templates later.
var driverResources = k8score.ResourceRequirements{
	Limits: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("0.5Gi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.5"),
	},
	Requests: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("64Mi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.1"),
	},
}

// Launcher only copies the binary into the volume, so it needs minimal resources.
var launcherResources = k8score.ResourceRequirements{
	Limits: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("128Mi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.5"),
	},
	Requests: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceCPU: k8sres.MustParse("0.1"),
	},
}
