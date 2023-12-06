/*
// Copyright 2023 kubeflow.org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/
package kfptask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"bytes"
	"strconv"

	"github.com/google/uuid"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/apis/kfptask"
	kfptaskv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/apis/kfptask/v1alpha1"
	kfptaskClient "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/client/clientset/versioned"
	kfptaskListers "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/client/listers/kfptask/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	listeners "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1"
	listenersv1beta1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/golang/protobuf/jsonpb"
)

// Check that our Reconciler implements Interface
var _ runreconciler.Interface = (*Reconciler)(nil)
var _ runreconciler.Finalizer = (*Reconciler)(nil)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	kubeClientSet     kubernetes.Interface
	pipelineClientSet clientset.Interface
	kfptaskClientSet  kfptaskClient.Interface
	runLister         listenersv1beta1.CustomRunLister
	kfptaskLister     kfptaskListers.KfpTaskLister
	taskRunLister     listeners.TaskRunLister
	clock             clock.RealClock
}

// borrow from kubeflow/pipeline/backend/src/common/util
const LabelKeyWorkflowRunId = "pipeline/runid"

// for kfptaskState and driver
const (
	StateInit = iota
	StateRunning
	StateDone
	// ReasonFailedValidation indicates that the reason for failure status is that Run failed runtime validation
	ReasonFailedValidation = "RunValidationFailed"

	// ReasonDriverError indicates that an error is throw while running the driver
	ReasonDriverError = "DriverError"

	// ReasonDriverSuccess indicates that driver finished successfully
	ReasonDriverSuccess = "DriverSuccess"
	ReasonTaskCached    = "TaskCached"

	ExecutionID    = "execution-id"
	ExecutorInput  = "executor-input"
	Condition      = "condition"
	IterationCount = "iteration-count"
)

type driverOptions struct {
	driverType  string
	options     driver.Options
	mlmdClient  *metadata.Client
	cacheClient *cacheutils.Client
}

type kfptaskState uint8

type kfptaskFS struct {
	ctx        context.Context
	reconciler *Reconciler
	run        *tektonv1beta1.CustomRun
	state      kfptaskState
	logger     *zap.SugaredLogger
}

// newReconciledNormal makes a new reconciler event with event type Normal, and reason RunReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(k8score.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", namespace, name)
}

func newKfpTaskFS(ctx context.Context, r *Reconciler, run *tektonv1beta1.CustomRun, logger *zap.SugaredLogger) *kfptaskFS {
	var state kfptaskState
	if !run.HasStarted() {
		state = StateInit
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the Run started %s", run.Namespace, run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
	} else if run.IsDone() || run.IsCancelled() {
		state = StateDone
	} else {
		state = StateRunning
	}
	return &kfptaskFS{ctx: ctx, reconciler: r, run: run, state: state, logger: logger}
}

// check if the run is in StateRunning or not
func (kts *kfptaskFS) isRunning() bool {
	return kts.state == StateRunning
}

// Not to propagate these labels to the spawned TaskRun
var labelsToDrop = map[string]string{
	"tekton.dev/pipelineRun":  "",
	"tekton.dev/pipelineTask": "",
	"tekton.dev/pipeline":     "",
	"tekton.dev/memberOf":     "",
	LabelKeyWorkflowRunId:     "",
}

var annotationToDrop = map[string]string{
	"kubectl.kubernetes.io/last-applied-configuration": "",
}

// transite to next state based on current state
func (kts *kfptaskFS) next(executionID string, executorInput string, podSpecPatch string) error {
	kts.logger.Infof("kts state is %s", kts.state)
	switch kts.state {
	case StateInit:
		// create the corresponding TaskRun CRD and start the task
		// compose TaskRun
		tr, err := kts.constructTaskRun(executionID, executorInput, podSpecPatch)
		if err != nil {
			kts.logger.Infof("Failed to construct a TaskRun:%v", err)
			kts.run.Status.MarkCustomRunFailed(kfptaskv1alpha1.KfpTaskRunReasonInternalError.String(), "Failed to construct a TaskRun: %v", err)
			return err
		}

		if _, err := kts.reconciler.pipelineClientSet.TektonV1().TaskRuns(kts.run.Namespace).Create(kts.ctx, tr, metav1.CreateOptions{}); err != nil {
			kts.run.Status.MarkCustomRunFailed(kfptaskv1alpha1.KfpTaskRunReasonFailed.String(), "Failed to create a TaskRun")
			return err
		}
		status := kfptaskv1alpha1.KfpTaskRunStatus{ChildTaskRun: tr.Name}
		if err := kts.run.Status.EncodeExtraFields(status); err != nil {
			// ignore the pipelinerun deletion error if there is
			kts.reconciler.deletePipelineRun(kts.ctx, kts.run.Namespace, tr.Name)
			kts.run.Status.MarkCustomRunFailed(kfptaskv1alpha1.KfpTaskRunReasonInternalError.String(), "Failed to encode extra fields")
			return err
		}
		kts.run.Status.MarkCustomRunRunning(kfptaskv1alpha1.KfpTaskRunReasonRunning.String(), "KfpTask kicks off a TaskRun:%s", tr.Name)
	case StateRunning:
		// check if underlying PipelineRun finishes or not
		status := kfptaskv1alpha1.KfpTaskRunStatus{}
		if err := kts.run.Status.DecodeExtraFields(&status); err != nil {
			kts.run.Status.MarkCustomRunFailed(kfptaskv1alpha1.KfpTaskRunReasonInternalError.String(), "Failed to decode extra fields")
			return err
		}

		tr, err := kts.reconciler.taskRunLister.TaskRuns(kts.run.Namespace).Get(status.ChildTaskRun)
		if err != nil {
			kts.run.Status.MarkCustomRunFailed(kfptaskv1alpha1.KfpTaskRunReasonInternalError.String(), "Failed to get the TaskRun: %s", status.ChildTaskRun)
			return err
		}

		condition := tr.Status.GetCondition(apis.ConditionSucceeded)
		if condition.IsTrue() {
			now := metav1.Now()
			kts.run.Status.CompletionTime = &now
			kts.run.Status.MarkCustomRunSucceeded(kfptaskv1alpha1.KfpTaskRunReasonSucceeded.String(), "KfpTask finished")
		} else {

			kts.run.Status.MarkCustomRunFailed(
				kfptaskv1alpha1.KfpTaskRunReasonFailed.String(),
				"the spawned TaskRun failed, TaskRun:%s, reason: %s, message: %s",
				status.ChildTaskRun, condition.Reason, condition.Message)
			return fmt.Errorf(
				"the spawned TaskRun failed, TaskRun:%s, reason: %s, message: %s",
				status.ChildTaskRun, condition.Reason, condition.Message)
		}

	case StateDone:
		// do nothing for this state, supposedly this method shouldn't be called while in StateDone
	}
	return nil
}

func (kts *kfptaskFS) constructTaskRun(executionID string, executorInput string, podSpecPatch string) (*tektonv1.TaskRun, error) {
	ktSpec, err := kts.reconciler.getKfpTaskSpec(kts.ctx, kts.run)
	if err != nil {
		return nil, err
	}

	pruuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	params := kts.run.Spec.Params
	// Pass execution ID to executor Input and task spec
	params = append(params, tektonv1beta1.Param{Name: ExecutionID, Value: tektonv1beta1.ParamValue{
		Type:      "string",
		StringVal: executionID,
	}})
	params = append(params, tektonv1beta1.Param{Name: ExecutorInput, Value: tektonv1beta1.ParamValue{
		Type:      "string",
		StringVal: executorInput,
	}})

	tr := &tektonv1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kt-%s", pruuid),
			Namespace: kts.run.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(kts.run,
				schema.GroupVersionKind{Group: pipeline.GroupName, Version: "v1beta1", Kind: "CustomRun"})},
			Labels:      dupStringMaps(kts.run.Labels, labelsToDrop),
			Annotations: dupStringMaps(kts.run.Annotations, annotationToDrop),
		},
		Spec: tektonv1.TaskRunSpec{
			Params:             v1ParamsConversion(context.Background(), params),
			Timeout:            kts.run.Spec.Timeout,
			ServiceAccountName: kts.run.Spec.ServiceAccountName,
			TaskSpec:           ktSpec.TaskSpec,
		},
	}

	if podSpecPatch != "" {
		podSpec, err := parseTaskSpecPatch(podSpecPatch)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TaskSpecPatch: %v", err)
		}
		taskSpec := tr.Spec.TaskSpec
		//TODO: need better merging strategy
		for p := range podSpec.Containers {
			if podSpec.Containers[p].Name == "main" {
				container := podSpec.Containers[p]
				// patch for the user-main step
				for i := range taskSpec.Steps {
					if taskSpec.Steps[i].Name == "user-main" {
						// merge the TaskSpec into this step
						taskSpec.Steps[i].Image = container.Image
						if len(container.Env) > 0 {
							taskSpec.Steps[i].Env = append(taskSpec.Steps[i].Env, container.Env...)
						}
						if len(container.EnvFrom) > 0 {
							taskSpec.Steps[i].EnvFrom = append(taskSpec.Steps[i].EnvFrom, container.EnvFrom...)
						}
						if len(container.VolumeMounts) > 0 {
							taskSpec.Steps[i].VolumeMounts = append(taskSpec.Steps[i].VolumeMounts, container.VolumeMounts...)
						}
						if len(container.Resources.Limits) > 0 || len(container.Resources.Requests) > 0 {
							taskSpec.Steps[i].ComputeResources = container.Resources
						}
						break
					}
				}
				break
			}
		}
		if len(podSpec.Volumes) > 0 {
			taskSpec.Volumes = append(taskSpec.Volumes, podSpec.Volumes...)
		}
		if len(podSpec.NodeSelector) > 0 {
			if tr.Spec.PodTemplate == nil {
				tr.Spec.PodTemplate = &pod.Template{}
			}
			for n, v := range podSpec.NodeSelector {
				tr.Spec.PodTemplate.NodeSelector[n] = v
			}
		}
	}

	return tr, nil
}

func parseTaskSpecPatch(taskSpecPatch string) (*k8score.PodSpec, error) {
	rev := k8score.PodSpec{}
	if err := json.Unmarshal([]byte(taskSpecPatch), &rev); err != nil {
		return nil, err
	}
	return &rev, nil
}

func dupStringMaps(source map[string]string, excludsive map[string]string) map[string]string {
	rev := make(map[string]string, len(source))
	for n, v := range source {
		if _, ok := excludsive[n]; !ok {
			rev[n] = v
		}
	}
	return rev
}

func (r *Reconciler) ReconcileKind(ctx context.Context, run *tektonv1beta1.CustomRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s", run.Namespace, run.Name)
	ktstate := newKfpTaskFS(ctx, r, run, logger)

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}
	if ktstate.isRunning() {
		return ktstate.next("", "", "")
	}
	options, err := parseParams(run)
	if err != nil {
		logger.Errorf("Run %s/%s is invalid because of %s", run.Namespace, run.Name, err)
		run.Status.MarkCustomRunFailed(ReasonFailedValidation,
			"Run can't be run because it has an invalid param - %v", err)
		return nil
	}

	runResults, runTask, executionID, executorInput, podSpecPatch, driverErr := execDriver(ctx, options)
	if driverErr != nil {
		logger.Errorf("kfp-driver execution failed when reconciling Run %s/%s: %v", run.Namespace, run.Name, driverErr)
		run.Status.MarkCustomRunFailed(ReasonDriverError,
			"kfp-driver execution failed: %v", driverErr)
		return nil
	}

	run.Status.Results = append(run.Status.Results, *runResults...)
	if !runTask {
		run.Status.MarkCustomRunSucceeded(ReasonDriverSuccess, "kfp-driver finished successfully")
		return newReconciledNormal(run.Namespace, run.Name)
	}

	if err := checkRefAndSpec(run); err != nil {
		logger.Infof("check error: %s", err.Error())
		return nil
	}

	return ktstate.next(executionID, executorInput, podSpecPatch)
}

func (r *Reconciler) FinalizeKind(ctx context.Context, run *tektonv1beta1.CustomRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Finalizing  %s/%s", run.Namespace, run.Name)

	if err := checkRefAndSpec(run); err != nil {
		logger.Info(err.Error())
		return nil
	}

	if !run.IsDone() {
		logger.Errorf("run: %s is not done yet but is deleted", run.Name)
	}

	// clean up spawned TaskRun if there is
	status := kfptaskv1alpha1.KfpTaskRunStatus{}
	if err := run.Status.DecodeExtraFields(&status); err != nil {
		logger.Debugf("KfpTask failed to decode the extra fields")
	}

	if status.ChildTaskRun != "" {
		if err := r.deletePipelineRun(ctx, run.Namespace, status.ChildTaskRun); err != nil {
			logger.Errorf("Failed to delete TaskRun:%s, error: %v", status.ChildTaskRun, err)
		}
	}

	logger.Debugf("Run is finished, done reconciling")
	return nil
}

func (r *Reconciler) deletePipelineRun(ctx context.Context, namespace, prName string) error {
	// TODO: for safty, may need to break the chain and check return value.
	return r.pipelineClientSet.TektonV1().PipelineRuns(namespace).Delete(ctx, prName, metav1.DeleteOptions{})
}

// double check run.Spec.Spec and run.Spec.Ref
func checkRefAndSpec(run *tektonv1beta1.CustomRun) error {
	if run.Spec.CustomRef != nil && run.Spec.CustomSpec != nil {
		return errors.New("contains both taskRef and taskSpec")
	}
	if run.Spec.CustomSpec == nil && run.Spec.CustomRef == nil {
		return errors.New("need taskRef or taskSpec")
	}

	// Check kind and apiVersion
	if run.Spec.CustomRef != nil &&
		(run.Spec.CustomRef.APIVersion != kfptaskv1alpha1.SchemeGroupVersion.String() || run.Spec.CustomRef.Kind != kfptask.Kind) {
		return fmt.Errorf("doesn't support %s/%s", run.Spec.CustomRef.APIVersion, run.Spec.CustomRef.Kind)
	}

	if run.Spec.CustomSpec != nil &&
		(run.Spec.CustomSpec.APIVersion != kfptaskv1alpha1.SchemeGroupVersion.String() || run.Spec.CustomSpec.Kind != kfptask.Kind) {
		return fmt.Errorf("doesn't support %s/%s", run.Spec.CustomSpec.APIVersion, run.Spec.CustomSpec.Kind)

	}
	return nil
}

func (r *Reconciler) getKfpTaskSpec(ctx context.Context, run *tektonv1beta1.CustomRun) (*kfptaskv1alpha1.KfpTaskSpec, error) {
	if run.Spec.CustomRef != nil {
		// retrieve taskRef of the ExitHandler
		kt, err := r.kfptaskClientSet.CustomV1alpha1().KfpTasks(run.Namespace).Get(ctx, run.Spec.CustomRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &kt.Spec, nil
	} else if run.Spec.CustomSpec != nil {
		var ktSpec kfptaskv1alpha1.KfpTaskSpec
		if err := json.Unmarshal(run.Spec.CustomSpec.Spec.Raw, &ktSpec); err != nil {
			return nil, err
		}
		return &ktSpec, nil
	}

	return nil, fmt.Errorf("run doesn't have taskRef or taskSpec")
}

func paramConvertTo(ctx context.Context, p *tektonv1beta1.Param, sink *tektonv1.Param) {
	sink.Name = p.Name
	newValue := tektonv1.ParamValue{}
	if p.Value.Type != "" {
		newValue.Type = tektonv1.ParamType(p.Value.Type)
	} else {
		newValue.Type = tektonv1.ParamType(tektonv1beta1.ParamTypeString)
	}
	newValue.StringVal = p.Value.StringVal
	newValue.ArrayVal = p.Value.ArrayVal
	newValue.ObjectVal = p.Value.ObjectVal
	sink.Value = newValue
}

func v1ParamsConversion(ctx context.Context, v1beta1Params tektonv1beta1.Params) []tektonv1.Param {
	v1Params := []tektonv1.Param{}
	for _, param := range v1beta1Params {
		v1Param := tektonv1.Param{}
		paramConvertTo(ctx, &param, &v1Param)
		v1Params = append(v1Params, v1Param)
	}
	return v1Params
}

func DAGPublisher(ctx context.Context, opts driver.Options, mlmd *metadata.Client) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish driver DAG execution %s: %w", fmt.Sprint(opts.DAGExecutionID), err)
		}
	}()
	var outputParameters map[string]*structpb.Value
	status := pb.Execution_COMPLETE
	execution, err := mlmd.GetExecution(ctx, opts.DAGExecutionID)
	if err = mlmd.PublishExecution(ctx, execution, outputParameters, nil, status); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	return nil
}

func execDriver(ctx context.Context, options *driverOptions) (*[]tektonv1beta1.CustomRunResult, bool, string, string, string, error) {
	var execution *driver.Execution
	var err error
	logger := logging.FromContext(ctx)
	taskRunDecision := false
	executionID := ""
	executorInput := ""
	podSpecPatch := ""

	switch options.driverType {
	case "ROOT_DAG":
		execution, err = driver.RootDAG(ctx, options.options, options.mlmdClient)
	case "CONTAINER":
		execution, err = driver.Container(ctx, options.options, options.mlmdClient, options.cacheClient)
	case "DAG":
		execution, err = driver.DAG(ctx, options.options, options.mlmdClient)
	case "DAG_PUB":
		// no-op for now
		err = DAGPublisher(ctx, options.options, options.mlmdClient)
		return &[]tektonv1beta1.CustomRunResult{}, taskRunDecision, executionID, executorInput, podSpecPatch, err
	default:
		err = fmt.Errorf("unknown driverType %s", options.driverType)
	}

	if err != nil {
		return nil, taskRunDecision, executionID, executorInput, podSpecPatch, err
	}

	var runResults []tektonv1beta1.CustomRunResult

	if execution.ID != 0 {
		logger.Infof("output execution.ID=%v", execution.ID)
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  ExecutionID,
			Value: fmt.Sprint(execution.ID),
		})
		executionID = fmt.Sprint(execution.ID)
	}
	if execution.IterationCount != nil {
		count := *execution.IterationCount
		// the count would be use as 'to' in PipelineLoop. since PipelineLoop's numberic iteration includes to,
		// need to substract 1 to compensate that.
		count = count - 1
		if count < 0 {
			count = 0
		}
		logger.Infof("output execution.IterationCount=%v, count:=%v", *execution.IterationCount, count)
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  IterationCount,
			Value: fmt.Sprint(count),
		})
	}

	logger.Infof("output execution.Condition=%v", execution.Condition)
	if execution.Condition == nil {
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  Condition,
			Value: "",
		})
	} else {
		runResults = append(runResults, tektonv1beta1.CustomRunResult{
			Name:  Condition,
			Value: strconv.FormatBool(*execution.Condition),
		})
	}

	if execution.ExecutorInput != nil {
		marshaler := jsonpb.Marshaler{}
		executorInputJSON, err := marshaler.MarshalToString(execution.ExecutorInput)
		if err != nil {
			return nil, taskRunDecision, executionID, executorInput, podSpecPatch, fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		logger.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
		executorInput = fmt.Sprint(executorInputJSON)
	}
	// seems no need to handle the PodSpecPatch
	if execution.Cached != nil {
		taskRunDecision = strconv.FormatBool(*execution.Cached) == "false"
	}

	if options.driverType == "CONTAINER" {
		podSpecPatch = execution.PodSpecPatch
	}

	return &runResults, taskRunDecision, executionID, executorInput, podSpecPatch, nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}

func parseParams(run *tektonv1beta1.CustomRun) (*driverOptions, *apis.FieldError) {
	if len(run.Spec.Params) == 0 {
		return nil, apis.ErrMissingField("params")
	}

	opts := &driverOptions{
		driverType: "",
	}
	opts.options.Namespace = run.Namespace
	mlmdOpts := metadata.ServerConfig{
		Address: "metadata-grpc-service.kubeflow.svc.cluster.local",
		Port:    "8080",
	}

	for _, param := range run.Spec.Params {
		switch param.Name {
		case "type":
			opts.driverType = param.Value.StringVal
		case "pipeline-name":
			opts.options.PipelineName = param.Value.StringVal
		case "run-id":
			opts.options.RunID = param.Value.StringVal
		case "component":
			if param.Value.StringVal != "" {
				componentSpec := &pipelinespec.ComponentSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, componentSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal component spec, error: %v\ncomponentSpec: %v", err, param.Value.StringVal),
						"component",
					)
				}
				opts.options.Component = componentSpec
			}
		case "task":
			if param.Value.StringVal != "" {
				taskSpec := &pipelinespec.PipelineTaskSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, taskSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal task spec, error: %v\ntask: %v", err, param.Value.StringVal),
						"task",
					)
				}
				opts.options.Task = taskSpec
			}
		case "runtime-config":
			if param.Value.StringVal != "" {
				runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, runtimeConfig); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal runtime config, error: %v\nruntimeConfig: %v", err, param.Value.StringVal),
						"runtime-config",
					)
				}
				opts.options.RuntimeConfig = runtimeConfig
			}
		case "container":
			if param.Value.StringVal != "" {
				containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, containerSpec); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal container spec, error: %v\ncontainerSpec: %v", err, param.Value.StringVal),
						"container",
					)
				}
				opts.options.Container = containerSpec
			}
		case "iteration-index":
			if param.Value.StringVal != "" {
				tmp, _ := strconv.ParseInt(param.Value.StringVal, 10, 32)
				opts.options.IterationIndex = int(tmp)
			} else {
				opts.options.IterationIndex = -1
			}
		case "dag-execution-id":
			if param.Value.StringVal != "" {
				opts.options.DAGExecutionID, _ = strconv.ParseInt(param.Value.StringVal, 10, 64)
			}
		case "mlmd-server-address":
			mlmdOpts.Address = param.Value.StringVal
		case "mlmd-server-port":
			mlmdOpts.Port = param.Value.StringVal
		case "kubernetes-config":
			var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
			if param.Value.StringVal != "" {
				k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
				if err := jsonpb.UnmarshalString(param.Value.StringVal, k8sExecCfg); err != nil {
					return nil, apis.ErrInvalidValue(
						fmt.Sprintf("failed to unmarshal Kubernetes config, error: %v\nKubernetesConfig: %v", err, param.Value.StringVal),
						"kubernetes-config",
					)
				}
				opts.options.KubernetesExecutorConfig = k8sExecCfg
			}
		}
	}

	//Check all options
	if opts.driverType == "" {
		return nil, apis.ErrMissingField("type")
	}

	if opts.options.RunID == "" {
		return nil, apis.ErrMissingField("run-id")
	}

	if opts.driverType == "ROOT_DAG" && opts.options.RuntimeConfig == nil {
		return nil, apis.ErrMissingField("runtime-config")
	}

	client, err := metadata.NewClient(mlmdOpts.Address, mlmdOpts.Port)
	if err != nil {
		return nil, apis.ErrGeneric(fmt.Sprintf("can't estibalish MLMD connection: %v", err))
	}
	opts.mlmdClient = client
	cacheClient, err := cacheutils.NewClient()
	if err != nil {
		return nil, apis.ErrGeneric(fmt.Sprintf("can't estibalish cache connection: %v", err))
	}
	opts.cacheClient = cacheClient
	return opts, nil
}
