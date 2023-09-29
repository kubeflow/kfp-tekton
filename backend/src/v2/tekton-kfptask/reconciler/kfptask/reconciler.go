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

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/apis/kfptask"
	kfptaskv1alpha1 "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/apis/kfptask/v1alpha1"
	kfptaskClient "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/client/clientset/versioned"
	kfptaskListers "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/client/listers/kfptask/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	listeners "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1"
	listenersv1beta1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
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

// for kfptaskState
const (
	StateInit = iota
	StateRunning
	StateDone
)

type kfptaskState uint8

type kfptaskFS struct {
	ctx        context.Context
	reconciler *Reconciler
	run        *tektonv1beta1.CustomRun
	state      kfptaskState
	logger     *zap.SugaredLogger
}

func newKfpTaskFS(ctx context.Context, r *Reconciler, run *tektonv1beta1.CustomRun, logger *zap.SugaredLogger) *kfptaskFS {
	var state kfptaskState
	if !run.HasStarted() {
		state = StateInit
		run.Status.InitializeConditions()
	} else if run.IsDone() || run.IsCancelled() {
		state = StateDone
	} else {
		state = StateRunning
	}
	return &kfptaskFS{ctx: ctx, reconciler: r, run: run, state: state, logger: logger}
}

// check if the run is in StateDone or not
func (kts *kfptaskFS) isDone() bool {
	return kts.state == StateDone
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
func (kts *kfptaskFS) next() error {
	switch kts.state {
	case StateInit:
		// create the corresponding TaskRun CRD and start the task
		// compose TaskRun
		tr, err := kts.constructTaskRun()
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

func (kts *kfptaskFS) constructTaskRun() (*tektonv1.TaskRun, error) {
	ktSpec, err := kts.reconciler.getKfpTaskSpec(kts.ctx, kts.run)
	if err != nil {
		return nil, err
	}

	pruuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	var podSpecPatch string
	params := kts.run.Spec.Params
	// get pod-spec-patch from params and remove it
	for idx, param := range kts.run.Spec.Params {
		if param.Name == "pod-spec-patch" {
			podSpecPatch = param.Value.StringVal
			params[idx] = kts.run.Spec.Params[len(kts.run.Spec.Params)-1]
			params = params[:len(params)-1]
			break
		}
	}

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
	logger.Infof("Reconciling %s/%s", run.Namespace, run.Name)

	if err := checkRefAndSpec(run); err != nil {
		logger.Infof("check error: %s", err.Error())
		return nil
	}

	ktstate := newKfpTaskFS(ctx, r, run, logger)
	// Ignore completed task.
	if ktstate.isDone() {
		logger.Debugf("Run is finished, done reconciling")
		return nil
	}

	return ktstate.next()
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

// func getJsonString(obj interface{}) string {
// 	raw, err := json.MarshalIndent(obj, "", "  ")
// 	if err != nil {
// 		return ""
// 	}
// 	return string(raw)
// }

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
