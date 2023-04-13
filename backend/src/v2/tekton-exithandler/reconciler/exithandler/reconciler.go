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
package exithandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis/exithandler"
	exithandlerv1alpha1 "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis/exithandler/v1alpha1"
	exithandlerClient "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client/clientset/versioned"
	exithandlerListers "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client/listers/exithandler/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	listeners "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
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
	kubeClientSet        kubernetes.Interface
	pipelineClientSet    clientset.Interface
	exitHandlerClientSet exithandlerClient.Interface
	runLister            listeners.CustomRunLister
	exitHandlerLister    exithandlerListers.ExitHandlerLister
	pipelineRunLister    listeners.PipelineRunLister
	clock                clock.RealClock
}

// borrow from kubeflow/pipeline/backend/src/common/util
const LabelKeyWorkflowRunId = "pipeline/runid"

// for exitHandlerState
const (
	StateInit = iota
	StateRunning
	StateDone
)

type exitHandlerState uint8

type exitHandlerFS struct {
	ctx        context.Context
	reconciler *Reconciler
	run        *tektonv1beta1.CustomRun
	state      exitHandlerState
	logger     *zap.SugaredLogger
}

func newExitHandlerFS(ctx context.Context, r *Reconciler, run *tektonv1beta1.CustomRun, logger *zap.SugaredLogger) *exitHandlerFS {
	var state exitHandlerState
	if !run.HasStarted() {
		state = StateInit
		run.Status.InitializeConditions()
	} else if run.IsDone() || run.IsCancelled() {
		state = StateDone
	} else {
		state = StateRunning
	}
	return &exitHandlerFS{ctx: ctx, reconciler: r, run: run, state: state, logger: logger}
}

// check if the run is in StateDone or not
func (ehs *exitHandlerFS) isDone() bool {
	return ehs.state == StateDone
}

// Not to propagate these labels to the spawned PipelineRun
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
func (ehs *exitHandlerFS) next() error {
	switch ehs.state {
	case StateInit:
		// create the corresponding PipelineRun CRD and start the pipeline
		// compose PipelineRun
		pr, err := ehs.constructPipelineRun()
		if err != nil {
			ehs.logger.Infof("Failed to construct a PipelineRun:%v", err)
			ehs.run.Status.MarkCustomRunFailed(exithandlerv1alpha1.ExitHandlerRunReasonInternalError.String(), "Failed to construct a PipelineRun: %v", err)
			return err
		}

		if _, err := ehs.reconciler.pipelineClientSet.TektonV1beta1().PipelineRuns(ehs.run.Namespace).Create(ehs.ctx, pr, metav1.CreateOptions{}); err != nil {
			ehs.run.Status.MarkCustomRunFailed(exithandlerv1alpha1.ExitHandlerRunReasonFailed.String(), "Failed to create a PipelineRun")
			return err
		}
		status := exithandlerv1alpha1.ExitHandlerRunStatus{ChildPipelineRun: pr.Name}
		if err := ehs.run.Status.EncodeExtraFields(status); err != nil {
			// ignore the pipelinerun deletion error if there is
			ehs.reconciler.deletePipelineRun(ehs.ctx, ehs.run.Namespace, pr.Name)
			ehs.run.Status.MarkCustomRunFailed(exithandlerv1alpha1.ExitHandlerRunReasonInternalError.String(), "Failed to encode extra fields")
			return err
		}
		ehs.run.Status.MarkCustomRunRunning(exithandlerv1alpha1.ExitHandlerRunReasonRunning.String(), "ExitHandler kicks off a PipelineRun:%s", pr.Name)
	case StateRunning:
		// check if underlying PipelineRun finishes or not
		status := exithandlerv1alpha1.ExitHandlerRunStatus{}
		if err := ehs.run.Status.DecodeExtraFields(&status); err != nil {
			ehs.run.Status.MarkCustomRunFailed(exithandlerv1alpha1.ExitHandlerRunReasonInternalError.String(), "Failed to decode extra fields")
			return err
		}

		pr, err := ehs.reconciler.pipelineClientSet.TektonV1beta1().PipelineRuns(ehs.run.Namespace).Get(ehs.ctx, status.ChildPipelineRun, metav1.GetOptions{})
		if err != nil {
			ehs.run.Status.MarkCustomRunFailed(exithandlerv1alpha1.ExitHandlerRunReasonInternalError.String(), "Failed to get the PipelineRun: %s", status.ChildPipelineRun)
			return err
		}

		condition := pr.Status.GetCondition(apis.ConditionSucceeded)
		if condition.IsTrue() {
			now := metav1.Now()
			ehs.run.Status.CompletionTime = &now
			ehs.run.Status.MarkCustomRunSucceeded(exithandlerv1alpha1.ExitHandlerRunReasonSucceeded.String(), "ExitHandler finished")
		} else {

			ehs.run.Status.MarkCustomRunFailed(
				exithandlerv1alpha1.ExitHandlerRunReasonFailed.String(),
				"the spawned PipelineRun failed, PipelineRun:%s, reason: %s, message: %s",
				status.ChildPipelineRun, condition.Reason, condition.Message)
			return fmt.Errorf(
				"the spawned PipelineRun failed, PipelineRun:%s, reason: %s, message: %s",
				status.ChildPipelineRun, condition.Reason, condition.Message)
		}

	case StateDone:
		// do nothing for this state, supposedly this method shouldn't be called while in StateDone
	}
	return nil
}

func (ehs *exitHandlerFS) constructPipelineRun() (*tektonv1beta1.PipelineRun, error) {
	ehSpec, err := ehs.reconciler.getExitHandlerSpec(ehs.ctx, ehs.run)
	if err != nil {
		return nil, err
	}

	pruuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	pr := &tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("eh-%s", pruuid),
			Namespace: ehs.run.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(ehs.run,
				schema.GroupVersionKind{Group: pipeline.GroupName, Version: "v1beta1", Kind: "CustomRun"})},
			Labels:      dupStringMaps(ehs.run.Labels, labelsToDrop),
			Annotations: dupStringMaps(ehs.run.Annotations, annotationToDrop),
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			Params:             ehs.run.Spec.Params,
			Timeout:            ehs.run.Spec.Timeout,
			ServiceAccountName: ehs.run.Spec.ServiceAccountName,
			PodTemplate:        ehSpec.PodTemplate,
			Workspaces:         ehs.run.Spec.Workspaces,
			PipelineSpec:       ehSpec.PipelineSpec,
		}}
	return pr, nil
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

	ehstate := newExitHandlerFS(ctx, r, run, logger)
	// Ignore completed task.
	if ehstate.isDone() {
		logger.Debugf("Run is finished, done reconciling")
		return nil
	}

	return ehstate.next()
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

	// clean up spawned PipelineRun if there is
	status := exithandlerv1alpha1.ExitHandlerRunStatus{}
	if err := run.Status.DecodeExtraFields(&status); err != nil {
		logger.Debugf("ExitHandler failed to decode the extra fields")
	}

	if status.ChildPipelineRun != "" {
		if err := r.deletePipelineRun(ctx, run.Namespace, status.ChildPipelineRun); err != nil {
			logger.Errorf("Failed to delete PipelineRun:%s, error: %v", status.ChildPipelineRun, err)
		}
	}

	logger.Debugf("Run is finished, done reconciling")
	return nil
}

func (r *Reconciler) deletePipelineRun(ctx context.Context, namespace, prName string) error {
	// TODO: for safty, may need to break the chain and check return value.
	return r.pipelineClientSet.TektonV1beta1().PipelineRuns(namespace).Delete(ctx, prName, metav1.DeleteOptions{})
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
		(run.Spec.CustomRef.APIVersion != exithandlerv1alpha1.SchemeGroupVersion.String() || run.Spec.CustomRef.Kind != exithandler.Kind) {
		return fmt.Errorf("doesn't support %s/%s", run.Spec.CustomRef.APIVersion, run.Spec.CustomRef.Kind)
	}

	if run.Spec.CustomSpec != nil &&
		(run.Spec.CustomSpec.APIVersion != exithandlerv1alpha1.SchemeGroupVersion.String() || run.Spec.CustomSpec.Kind != exithandler.Kind) {
		return fmt.Errorf("doesn't support %s/%s", run.Spec.CustomSpec.APIVersion, run.Spec.CustomSpec.Kind)

	}
	return nil
}

func (r *Reconciler) getExitHandlerSpec(ctx context.Context, run *tektonv1beta1.CustomRun) (*exithandlerv1alpha1.ExitHandlerSpec, error) {
	if run.Spec.CustomRef != nil {
		// retrieve taskRef of the ExitHandler
		eh, err := r.exitHandlerClientSet.CustomV1alpha1().ExitHandlers(run.Namespace).Get(ctx, run.Spec.CustomRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &eh.Spec, nil
	} else if run.Spec.CustomSpec != nil {
		var ehSpec exithandlerv1alpha1.ExitHandlerSpec
		if err := json.Unmarshal(run.Spec.CustomSpec.Spec.Raw, &ehSpec); err != nil {
			return nil, err
		}
		return &ehSpec, nil
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
