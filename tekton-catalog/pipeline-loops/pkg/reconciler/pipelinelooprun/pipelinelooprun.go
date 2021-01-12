/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipelinelooprun

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	pipelineloopclientset "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/clientset/versioned"
	listerspipelineloop "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/listers/pipelineloop/v1alpha1"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// pipelineLoopLabelKey is the label identifier for a PipelineLoop.  This label is added to the Run and its PipelineRuns.
	pipelineLoopLabelKey = "/pipelineLoop"

	// pipelineLoopRunLabelKey is the label identifier for a Run.  This label is added to the Run's PipelineRuns.
	pipelineLoopRunLabelKey = "/run"

	// pipelineLoopIterationLabelKey is the label identifier for the iteration number.  This label is added to the Run's PipelineRuns.
	pipelineLoopIterationLabelKey = "/pipelineLoopIteration"
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	pipelineClientSet     clientset.Interface
	pipelineloopClientSet pipelineloopclientset.Interface
	runLister             listersalpha.RunLister
	pipelineLoopLister    listerspipelineloop.PipelineLoopLister
	pipelineRunLister     listers.PipelineRunLister
}

var (
	// Check that our Reconciler implements runreconciler.Interface
	_ runreconciler.Interface = (*Reconciler)(nil)
)

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the Run resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())

	// Check that the Run references a PipelineLoop CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	if run.Spec.Ref == nil ||
		run.Spec.Ref.APIVersion != pipelineloopv1alpha1.SchemeGroupVersion.String() ||
		run.Spec.Ref.Kind != pipelineloop.PipelineLoopControllerName {
		logger.Errorf("Received control for a Run %s/%s that does not reference a PipelineLoop custom CRD", run.Namespace, run.Name)
		return nil
	}

	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s createTimestamp %s is after the Run started %s", run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
		// Emit events. During the first reconcile the status of the Run may change twice
		// from not Started to Started and then to Running, so we need to sent the event here
		// and at the end of 'Reconcile' again.
		// We also want to send the "Started" event as soon as possible for anyone who may be waiting
		// on the event to perform user facing initialisations, such has reset a CI check status
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	// Store the condition before reconcile
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	// Reconcile the Run
	if err := c.reconcile(ctx, run, status); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
	}

	if err := c.updateLabelsAndAnnotations(ctx, run); err != nil {
		logger.Warn("Failed to update Run labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	if err := run.Status.EncodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonInternalError.String(),
			"Internal error calling EncodeExtraFields: %v", err)
		logger.Errorf("EncodeExtraFields error: %v", err.Error())
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	// Only transient errors that should retry the reconcile are returned.
	return merr
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *pipelineloopv1alpha1.PipelineLoopRunStatus) error {
	logger := logging.FromContext(ctx)

	// Get the PipelineLoop referenced by the Run
	pipelineLoopMeta, pipelineLoopSpec, err := c.getPipelineLoop(ctx, run)
	if err != nil {
		return nil
	}

	// Store the fetched PipelineLoopSpec on the Run for auditing
	storePipelineLoopSpec(status, pipelineLoopSpec)

	// Propagate labels and annotations from PipelineLoop to Run.
	propagatePipelineLoopLabelsAndAnnotations(run, pipelineLoopMeta)

	// Validate PipelineLoop spec
	if err := pipelineLoopSpec.Validate(ctx); err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"PipelineLoop %s/%s can't be Run; it has an invalid spec: %s",
			pipelineLoopMeta.Namespace, pipelineLoopMeta.Name, err)
		return nil
	}

	// Determine how many iterations of the Task will be done.
	totalIterations, err := computeIterations(run, pipelineLoopSpec)
	if err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"Cannot determine number of iterations: %s", err)
		return nil
	}

	// Update status of PipelineRuns.  Return the PipelineRun representing the highest loop iteration.
	highestIteration, highestIterationPr, err := c.updatePipelineRunStatus(logger, run, status)
	if err != nil {
		return fmt.Errorf("error updating PipelineRun status for Run %s/%s: %w", run.Namespace, run.Name, err)
	}

	// Check the status of the PipelineRun for the highest iteration.
	if highestIterationPr != nil {
		// If it's not done, wait for it to finish or cancel it if the run is cancelled.
		if !highestIterationPr.IsDone() {
			if run.IsCancelled() {
				logger.Infof("Run %s/%s is cancelled.  Cancelling PipelineRun %s.", run.Namespace, run.Name, highestIterationPr.Name)
				b, err := getCancelPatch()
				if err != nil {
					return fmt.Errorf("Failed to make patch to cancel PipelineRun %s: %v", highestIterationPr.Name, err)
				}
				if _, err := c.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Patch(ctx, highestIterationPr.Name, types.JSONPatchType, b, metav1.PatchOptions{}); err != nil {
					run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntCancel.String(),
						"Failed to patch PipelineRun `%s` with cancellation: %v", highestIterationPr.Name, err)
					return nil
				}
				// Update status. It is still running until the PipelineRun is actually cancelled.
				run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
					"Cancelling PipelineRun %s", highestIterationPr.Name)
				return nil
			}
			run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
				"Iterations completed: %d", highestIteration-1)
			return nil
		}

		if !highestIterationPr.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
			if run.IsCancelled() {
				run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String(),
					"Run %s/%s was cancelled",
					run.Namespace, run.Name)
			} else {
				run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailed.String(),
					"PipelineRun %s has failed", highestIterationPr.Name)
			}
			return nil
		}

		// Mark run successful if the condition are met.
		// if the last loop task is skipped, but the highestIterationPr successed. Mark run success.
		// lastLoopTask := highestIterationPr.ObjectMeta.Annotations["last-loop-task"]
		lastLoopTask := ""
		for key, val := range run.ObjectMeta.Labels {
			if key == "last-loop-task" {
				lastLoopTask = val
			}
		}
		if lastLoopTask != "" {
			skippedTaskList := highestIterationPr.Status.SkippedTasks
			for _, task := range skippedTaskList {
				if task.Name == lastLoopTask {
					// Mark run successful and stop the loop pipelinerun
					run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					run.Status.Results = []v1beta1.TaskRunResult{{
						Name:  "condition",
						Value: "pass",
					}}
					return nil
				}
			}
		}
	}

	// Move on to the next iteration (or the first iteration if there was no PipelineRun).
	// Check if the Run is done.
	nextIteration := highestIteration + 1
	if nextIteration > totalIterations {
		run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
			"All PipelineRuns completed successfully")
		run.Status.Results = []v1beta1.TaskRunResult{{
			Name:  "condition",
			Value: "fail",
		}}
		return nil
	}
	// Before starting up another PipelineRun, check if the run was cancelled.
	if run.IsCancelled() {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String(),
			"Run %s/%s was cancelled",
			run.Namespace, run.Name)
		return nil
	}

	// Create a PipelineRun to run this iteration.
	pr, err := c.createPipelineRun(ctx, logger, pipelineLoopSpec, run, nextIteration)
	if err != nil {
		return fmt.Errorf("error creating PipelineRun from Run %s: %w", run.Name, err)
	}

	status.PipelineRuns[pr.Name] = &pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{
		Iteration: nextIteration,
		Status:    &pr.Status,
	}

	run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
		"Iterations completed: %d", highestIteration)

	return nil
}

func (c *Reconciler) getPipelineLoop(ctx context.Context, run *v1alpha1.Run) (*metav1.ObjectMeta, *pipelineloopv1alpha1.PipelineLoopSpec, error) {
	pipelineLoopMeta := metav1.ObjectMeta{}
	pipelineLoopSpec := pipelineloopv1alpha1.PipelineLoopSpec{}
	if run.Spec.Ref != nil && run.Spec.Ref.Name != "" {
		// Use the k8 client to get the PipelineLoop rather than the lister.  This avoids a timing issue where
		// the PipelineLoop is not yet in the lister cache if it is created at nearly the same time as the Run.
		// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
		//
		// tl, err := c.pipelineLoopLister.PipelineLoops(run.Namespace).Get(run.Spec.Ref.Name)
		tl, err := c.pipelineloopClientSet.CustomV1alpha1().PipelineLoops(run.Namespace).Get(ctx, run.Spec.Ref.Name, metav1.GetOptions{})
		if err != nil {
			run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
				"Error retrieving PipelineLoop for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("Error retrieving PipelineLoop for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		pipelineLoopMeta = tl.ObjectMeta
		pipelineLoopSpec = tl.Spec
	} else {
		// Run does not require name but for PipelineLoop it does.
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
			"Missing spec.ref.name for Run %s/%s",
			run.Namespace, run.Name)
		return nil, nil, fmt.Errorf("Missing spec.ref.name for Run %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name))
	}
	return &pipelineLoopMeta, &pipelineLoopSpec, nil
}

func (c *Reconciler) createPipelineRun(ctx context.Context, logger *zap.SugaredLogger, tls *pipelineloopv1alpha1.PipelineLoopSpec, run *v1alpha1.Run, iteration int) (*v1beta1.PipelineRun, error) {

	// Create name for PipelineRun from Run name plus iteration number.
	prName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", run.Name, fmt.Sprintf("%05d", iteration)))

	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            prName,
			Namespace:       run.Namespace,
			OwnerReferences: []metav1.OwnerReference{run.GetOwnerReference()},
			Labels:          getPipelineRunLabels(run, strconv.Itoa(iteration)),
			Annotations:     getPipelineRunAnnotations(run),
		},
		Spec: v1beta1.PipelineRunSpec{
			Params:             getParameters(run, tls, iteration),
			Timeout:            tls.Timeout,
			ServiceAccountName: "",  // TODO: Implement service account name
			PodTemplate:        nil, // TODO: Implement pod template
		}}

	if tls.PipelineRef != nil {
		pr.Spec.PipelineRef = &v1beta1.PipelineRef{
			Name: tls.PipelineRef.Name,
			// Kind: tls.PipelineRef.Kind,
		}
	} else if tls.PipelineSpec != nil {
		pr.Spec.PipelineSpec = tls.PipelineSpec
	}

	logger.Infof("Creating a new PipelineRun object %s", prName)
	return c.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Create(ctx, pr, metav1.CreateOptions{})

}

// func (c *Reconciler) retryPipelineRun(ctx context.Context, tr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
// 	newStatus := *tr.Status.DeepCopy()
// 	newStatus.RetriesStatus = nil
// 	tr.Status.RetriesStatus = append(tr.Status.RetriesStatus, newStatus)
// 	tr.Status.StartTime = nil
// 	tr.Status.CompletionTime = nil
// 	tr.Status.PodName = ""
// 	tr.Status.SetCondition(&apis.Condition{
// 		Type:   apis.ConditionSucceeded,
// 		Status: corev1.ConditionUnknown,
// 	})
// 	return c.pipelineClientSet.TektonV1beta1().PipelineRuns(tr.Namespace).UpdateStatus(ctx, tr, metav1.UpdateOptions{})
// }

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1alpha1.Run) error {
	newRun, err := c.runLister.Runs(run.Namespace).Get(run.Name)
	if err != nil {
		return fmt.Errorf("error getting Run %s when updating labels/annotations: %w", run.Name, err)
	}
	if !reflect.DeepEqual(run.ObjectMeta.Labels, newRun.ObjectMeta.Labels) || !reflect.DeepEqual(run.ObjectMeta.Annotations, newRun.ObjectMeta.Annotations) {
		mergePatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels":      run.ObjectMeta.Labels,
				"annotations": run.ObjectMeta.Annotations,
			},
		}
		patch, err := json.Marshal(mergePatch)
		if err != nil {
			return err
		}
		_, err = c.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).Patch(ctx, run.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}

func (c *Reconciler) updatePipelineRunStatus(logger *zap.SugaredLogger, run *v1alpha1.Run, status *pipelineloopv1alpha1.PipelineLoopRunStatus) (int, *v1beta1.PipelineRun, error) {
	highestIteration := 0
	var highestIterationPr *v1beta1.PipelineRun = nil
	if status.PipelineRuns == nil {
		status.PipelineRuns = make(map[string]*pipelineloopv1alpha1.PipelineLoopPipelineRunStatus)
	}
	pipelineRunLabels := getPipelineRunLabels(run, "")
	pipelineRuns, err := c.pipelineRunLister.PipelineRuns(run.Namespace).List(labels.SelectorFromSet(pipelineRunLabels))
	if err != nil {
		return 0, nil, fmt.Errorf("could not list PipelineRuns %#v", err)
	}
	if pipelineRuns == nil || len(pipelineRuns) == 0 {
		return 0, nil, nil
	}
	for _, pr := range pipelineRuns {
		lbls := pr.GetLabels()
		iterationStr := lbls[pipelineloop.GroupName+pipelineLoopIterationLabelKey]
		iteration, err := strconv.Atoi(iterationStr)
		if err != nil {
			run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
				"Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			logger.Errorf("Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			return 0, nil, nil
		}
		status.PipelineRuns[pr.Name] = &pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{
			Iteration: iteration,
			Status:    &pr.Status,
		}
		if iteration > highestIteration {
			highestIteration = iteration
			highestIterationPr = pr
		}
	}
	return highestIteration, highestIterationPr, nil
}

func getCancelPatch() ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.PipelineRunSpecStatusCancelled,
	}}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch bytes in order to cancel: %v", err)
	}
	return patchBytes, nil
}

func computeIterations(run *v1alpha1.Run, tls *pipelineloopv1alpha1.PipelineLoopSpec) (int, error) {
	// Find the iterate parameter.
	numberOfIterations := -1
	from := -1
	step := -1
	to := -1
	for _, p := range run.Spec.Params {
		if tls.IterateNumeric != "" {
			if p.Name == "from" {
				from, _ = strconv.Atoi(p.Value.StringVal)
			}
			if p.Name == "step" {
				step, _ = strconv.Atoi(p.Value.StringVal)
			}
			if p.Name == "to" {
				to, _ = strconv.Atoi(p.Value.StringVal)
			}
		} else if p.Name == tls.IterateParam {
			if p.Value.Type == v1beta1.ParamTypeString {
				// Transfer p.Value to Array.
				var strings []string
				if err := json.Unmarshal([]byte(p.Value.StringVal), &strings); err != nil {
					return 0, fmt.Errorf("The value of the iterate parameter %q can not transfer to array", tls.IterateParam)
				}
				numberOfIterations = len(strings)
				break
			}
			if p.Value.Type == v1beta1.ParamTypeArray {
				numberOfIterations = len(p.Value.ArrayVal)
				break
			}
		}
	}
	if tls.IterateNumeric != "" {
		if from == -1 || to == -1 {
			return 0, fmt.Errorf("The from or to parameters was not found in runs")
		}
		if step == -1 {
			step = 1
		}
		// numberOfIterations is the number of (to - from) / step + 1
		numberOfIterations = (to-from)/step + 1
	} else if numberOfIterations == -1 {
		return 0, fmt.Errorf("The iterate parameter %q was not found", tls.IterateParam)
	}
	return numberOfIterations, nil
}

func getParameters(run *v1alpha1.Run, tls *pipelineloopv1alpha1.PipelineLoopSpec, iteration int) []v1beta1.Param {
	var out []v1beta1.Param
	if tls.IterateParam != "" {
		// IterateParam defined
		for i, p := range run.Spec.Params {
			if p.Name == tls.IterateParam {
				if p.Value.Type == v1beta1.ParamTypeArray {
					out = append(out, v1beta1.Param{
						Name:  p.Name,
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: p.Value.ArrayVal[iteration-1]},
					})
				}
				if p.Value.Type == v1beta1.ParamTypeString {
					var strings []string
					_ = json.Unmarshal([]byte(p.Value.StringVal), &strings)
					out = append(out, v1beta1.Param{
						Name:  p.Name,
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: strings[iteration-1]},
					})
				}
			} else {
				out = append(out, run.Spec.Params[i])
			}
		}
	} else {
		// IterateNumeric defined
		IterateStrings := []string{"from", "step", "to"}
		for i, p := range run.Spec.Params {
			if _, found := Find(IterateStrings, p.Name); !found {
				out = append(out, run.Spec.Params[i])
			}
		}
		out = append(out, v1beta1.Param{
			Name:  tls.IterateNumeric,
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: strconv.Itoa(iteration)},
		})
	}
	return out
}

func getPipelineRunAnnotations(run *v1alpha1.Run) map[string]string {
	// Propagate annotations from Run to PipelineRun.
	annotations := make(map[string]string, len(run.ObjectMeta.Annotations)+1)
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func getPipelineRunLabels(run *v1alpha1.Run, iterationStr string) map[string]string {
	// Propagate labels from Run to PipelineRun.
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	ignnoreLabelsKey := []string{"tekton.dev/pipelineRun", "tekton.dev/pipelineTask", "tekton.dev/pipeline"}
	for key, val := range run.ObjectMeta.Labels {
		if _, found := Find(ignnoreLabelsKey, key); !found {
			labels[key] = val
		}
	}
	// Note: The Run label uses the normal Tekton group name.
	labels[pipeline.GroupName+pipelineLoopRunLabelKey] = run.Name
	if iterationStr != "" {
		labels[pipelineloop.GroupName+pipelineLoopIterationLabelKey] = iterationStr
	}
	return labels
}

func propagatePipelineLoopLabelsAndAnnotations(run *v1alpha1.Run, pipelineLoopMeta *metav1.ObjectMeta) {
	// Propagate labels from PipelineLoop to Run.
	if run.ObjectMeta.Labels == nil {
		run.ObjectMeta.Labels = make(map[string]string, len(pipelineLoopMeta.Labels)+1)
	}
	for key, value := range pipelineLoopMeta.Labels {
		run.ObjectMeta.Labels[key] = value
	}
	run.ObjectMeta.Labels[pipelineloop.GroupName+pipelineLoopLabelKey] = pipelineLoopMeta.Name

	// Propagate annotations from PipelineLoop to Run.
	if run.ObjectMeta.Annotations == nil {
		run.ObjectMeta.Annotations = make(map[string]string, len(pipelineLoopMeta.Annotations))
	}
	for key, value := range pipelineLoopMeta.Annotations {
		run.ObjectMeta.Annotations[key] = value
	}
}

func storePipelineLoopSpec(status *pipelineloopv1alpha1.PipelineLoopRunStatus, tls *pipelineloopv1alpha1.PipelineLoopSpec) {
	// Only store the PipelineLoopSpec once, if it has never been set before.
	if status.PipelineLoopSpec == nil {
		status.PipelineLoopSpec = tls
	}
}
