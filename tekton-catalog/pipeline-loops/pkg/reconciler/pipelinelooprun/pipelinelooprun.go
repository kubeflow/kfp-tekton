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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/hashicorp/go-multierror"
	cache "github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/model"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	pipelineloopclientset "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/clientset/versioned"
	listerspipelineloop "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/listers/pipelineloop/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	runv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/run/v1alpha1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	tkstatus "github.com/tektoncd/pipeline/pkg/status"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// pipelineLoopLabelKey is the label identifier for a PipelineLoop.  This label is added to the Run and its PipelineRuns.
	pipelineLoopLabelKey = "/pipelineLoop"

	// pipelineLoopRunLabelKey is the label identifier for a Run.  This label is added to the Run's PipelineRuns.
	pipelineLoopRunLabelKey = "/run"

	// parentPRKey is the label identifier for the original Pipelnerun who created the Run.  This label is added to the Run's PipelineRuns.
	parentPRKey = "/parentPipelineRun"

	// originalPRKey is the label identifier for the original Pipelnerun (first Pipelinerun)
	originalPRKey = "/originalPipelineRun"

	// pipelineLoopIterationLabelKey is the label identifier for the iteration number.  This label is added to the Run's PipelineRuns.
	pipelineLoopIterationLabelKey = "/pipelineLoopIteration"

	// pipelineLoopCurrentIterationItemAnnotationKey is the annotation identifier for the iteration items (string array). This annotation is added to the Run's PipelineRuns.
	pipelineLoopCurrentIterationItemAnnotationKey = "/pipelineLoopCurrentIterationItem"

	// LabelKeyWorkflowRunId is the label identifier a pipelinerun is managed by the Kubeflow Pipeline persistent agent.
	LabelKeyWorkflowRunId = "pipeline/runid"

	DefaultNestedStackDepth = 30
	DefaultIterationLimit   = 10000

	MaxNestedStackDepthKey = "maxNestedStackDepth"
	IterationLimitEnvKey   = "IterationLimit"

	defaultIterationParamStrSeparator = ","
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	KubeClientSet         kubernetes.Interface
	pipelineClientSet     clientset.Interface
	pipelineloopClientSet pipelineloopclientset.Interface
	runLister             listersalpha.RunLister
	pipelineLoopLister    listerspipelineloop.PipelineLoopLister
	pipelineRunLister     listers.PipelineRunLister
	cacheStore            *cache.TaskCacheStore
	clock                 clock.RealClock
}
type CacheKey struct {
	Params           []v1beta1.Param                        `json:"params"`
	PipelineLoopSpec *pipelineloopv1alpha1.PipelineLoopSpec `json:"pipelineSpec"`
}

var (
	// Check that our Reconciler implements runreconciler.Interface
	_                runreconciler.Interface = (*Reconciler)(nil)
	cancelPatchBytes []byte
	iterationLimit   int = DefaultIterationLimit
)

func init() {
	var err error
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.PipelineRunSpecStatusCancelled,
	}}
	cancelPatchBytes, err = json.Marshal(patches)
	if err != nil {
		log.Fatalf("failed to marshal patch bytes in order to cancel: %v", err)
	}
	iterationLimitEnv, ok := os.LookupEnv(IterationLimitEnvKey)
	if ok {
		iterationLimitNum, err := strconv.Atoi(iterationLimitEnv)
		if err == nil {
			iterationLimit = iterationLimitNum
		}
	}
}

func isCachingEnabled(run *v1alpha1.Run) bool {
	return run.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] == "true"
}

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the Run resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())
	if run.Spec.Ref != nil && run.Spec.Spec != nil {
		logger.Errorf("Run %s/%s can provide one of Run.Spec.Ref/Run.Spec.Spec", run.Namespace, run.Name)
		return nil
	}
	if run.Spec.Spec == nil && run.Spec.Ref == nil {
		logger.Errorf("Run %s/%s does not provide a spec or ref.", run.Namespace, run.Name)
		return nil
	}
	if (run.Spec.Ref != nil && run.Spec.Ref.Kind == pipelineloop.BreakTaskName) ||
		(run.Spec.Spec != nil && run.Spec.Spec.Kind == pipelineloop.BreakTaskName) {
		if !run.IsDone() {
			run.Status.InitializeConditions()
			run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
				"Break task is a dummy task.")
		}
		logger.Infof("Break task encountered %s", run.Name)
		return nil
	}
	// Check that the Run references a PipelineLoop CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	if run.Spec.Ref != nil &&
		(run.Spec.Ref.APIVersion != pipelineloopv1alpha1.SchemeGroupVersion.String() ||
			run.Spec.Ref.Kind != pipelineloop.PipelineLoopControllerName) {
		logger.Errorf("Received control for a Run %s/%s/%v that does not reference a PipelineLoop custom CRD ref", run.Namespace, run.Name, run.Spec.Ref)
		return nil
	}

	if run.Spec.Spec != nil &&
		(run.Spec.Spec.APIVersion != pipelineloopv1alpha1.SchemeGroupVersion.String() ||
			run.Spec.Spec.Kind != pipelineloop.PipelineLoopControllerName) {
		logger.Errorf("Received control for a Run %s/%s that does not reference a PipelineLoop custom CRD spec", run.Namespace, run.Name)
		return nil
	}
	logger.Infof("Received control for a Run %s/%s %-v", run.Namespace, run.Name, run.Spec.Spec)
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

	// Store the condition before reconcile
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	if run.IsDone() {
		if run.IsSuccessful() && !c.cacheStore.Disabled && isCachingEnabled(run) {
			marshal, err := json.Marshal(CacheKey{
				PipelineLoopSpec: status.PipelineLoopSpec,
				Params:           run.Spec.Params,
			})
			if err == nil {
				hashSum := fmt.Sprintf("%x", md5.Sum(marshal))
				resultBytes, err1 := json.Marshal(run.Status.Results)
				if err1 != nil {
					return fmt.Errorf("error while marshalling result to cache for run: %s, %w", run.Name, err)
				}
				err = c.cacheStore.Put(&model.TaskCache{
					TaskHashKey: hashSum,
					TaskOutput:  string(resultBytes),
				})
				if err != nil {
					return fmt.Errorf("error while adding result to cache for run: %s, %w", run.Name, err)
				}
				logger.Infof("cached the results of successful run %s, with key: %s", run.Name, hashSum)
			}
		}
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
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
	return merr
}

func EnableCustomTaskFeatureFlag(ctx context.Context) context.Context {
	defaults, _ := config.NewDefaultsFromMap(map[string]string{})
	featureFlags, _ := config.NewFeatureFlagsFromMap(map[string]string{
		"enable-custom-tasks": "true",
	})
	artifactBucket, _ := config.NewArtifactBucketFromMap(map[string]string{})
	artifactPVC, _ := config.NewArtifactPVCFromMap(map[string]string{})
	c := &config.Config{
		Defaults:       defaults,
		FeatureFlags:   featureFlags,
		ArtifactBucket: artifactBucket,
		ArtifactPVC:    artifactPVC,
	}
	return config.ToContext(ctx, c)
}

// Check if this pipelineLoop starts other pipelineLoop(s)
func isNestedPipelineLoop(pipelineLoopSpec *pipelineloopv1alpha1.PipelineLoopSpec) bool {
	if pipelineLoopSpec.PipelineSpec == nil {
		return false
	}
	for _, t := range pipelineLoopSpec.PipelineSpec.Tasks {
		if t.TaskSpec != nil {
			if t.TaskSpec.Kind == "PipelineLoop" {
				return true
			}
		} else if t.TaskRef != nil {
			if t.TaskRef.Kind == "PipelineLoop" {
				return true
			}
		}
	}
	return false
}

func getMaxNestedStackDepth(pipelineLoopMeta *metav1.ObjectMeta) (int, error) {
	maxNestedStackDepth := pipelineLoopMeta.Annotations[MaxNestedStackDepthKey]
	if maxNestedStackDepth != "" {
		atoi, err := strconv.Atoi(maxNestedStackDepth)
		return atoi, err
	}
	return DefaultNestedStackDepth, nil
}

func (c *Reconciler) setMaxNestedStackDepth(ctx context.Context, pipelineLoopSpec *pipelineloopv1alpha1.PipelineLoopSpec, run *v1alpha1.Run, depth int) {
	logger := logging.FromContext(ctx)

	if pipelineLoopSpec.PipelineSpec == nil {
		return
	}
	for k, t := range pipelineLoopSpec.PipelineSpec.Tasks {
		if t.TaskSpec != nil {
			if t.TaskSpec.Kind == "PipelineLoop" {
				if len(t.TaskSpec.Metadata.Annotations) == 0 {
					t.TaskSpec.Metadata.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
				} else {
					t.TaskSpec.Metadata.Annotations[MaxNestedStackDepthKey] = fmt.Sprint(depth)
				}
				pipelineLoopSpec.PipelineSpec.Tasks[k].TaskSpec.Metadata.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
			}
		} else if t.TaskRef != nil {
			if t.TaskRef.Kind == "PipelineLoop" {
				tl, err := c.pipelineloopClientSet.CustomV1alpha1().PipelineLoops(run.Namespace).Get(ctx, t.TaskRef.Name, metav1.GetOptions{})
				if err == nil && tl != nil {
					if len(tl.ObjectMeta.Annotations) == 0 {
						tl.ObjectMeta.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
					} else {
						tl.ObjectMeta.Annotations[MaxNestedStackDepthKey] = fmt.Sprint(depth)
					}
					_, err := c.pipelineloopClientSet.CustomV1alpha1().PipelineLoops(run.Namespace).Update(ctx, tl, metav1.UpdateOptions{})
					if err != nil {
						logger.Errorf("Error while updating pipelineloop nested stack depth, %v", err)
					}
				} else if err != nil {
					logger.Warnf("Unable to fetch pipelineLoop wiht name: %s error: %v", t.TaskRef.Name, err)
				}
			}
		}
	}
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *pipelineloopv1alpha1.PipelineLoopRunStatus) error {
	ctx = EnableCustomTaskFeatureFlag(ctx)
	logger := logging.FromContext(ctx)
	var hashSum string
	// Get the PipelineLoop referenced by the Run
	pipelineLoopMeta, pipelineLoopSpec, err := c.getPipelineLoop(ctx, run)
	if err != nil {
		return nil
	}
	// Store the fetched PipelineLoopSpec on the Run for auditing
	storePipelineLoopSpec(status, pipelineLoopSpec)

	// Propagate labels and annotations from PipelineLoop to Run.
	propagatePipelineLoopLabelsAndAnnotations(run, pipelineLoopMeta)

	pipelineLoopSpec.SetDefaults(ctx)
	// Validate PipelineLoop spec
	if err := pipelineLoopSpec.Validate(ctx); err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"PipelineLoop %s/%s can't be Run; it has an invalid spec: %s",
			pipelineLoopMeta.Namespace, pipelineLoopMeta.Name, err)
		return nil
	}

	// Determine how many iterations of the Task will be done.
	totalIterations, iterationElements, err := computeIterations(run, pipelineLoopSpec)
	if err != nil {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"Cannot determine number of iterations: %s", err)
		return nil
	}
	if totalIterations > iterationLimit {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"Total number of iterations exceeds the limit: %d", iterationLimit)
		return nil
	}

	// Update status of PipelineRuns.  Return the PipelineRun representing the highest loop iteration.
	highestIteration, currentRunningPrs, failedPrs, err := c.updatePipelineRunStatus(ctx, iterationElements, run, status)
	if err != nil {
		return fmt.Errorf("error updating PipelineRun status for Run %s/%s: %w", run.Namespace, run.Name, err)
	}
	if !c.cacheStore.Disabled && isCachingEnabled(run) {
		marshal, err := json.Marshal(CacheKey{
			PipelineLoopSpec: pipelineLoopSpec,
			Params:           run.Spec.Params,
		})
		if marshal != nil && err == nil {
			hashSum = fmt.Sprintf("%x", md5.Sum(marshal))
			taskCache, err := c.cacheStore.Get(hashSum)
			if err == nil && taskCache != nil {
				logger.Infof("Found a cached entry, for run: %s, with key:", run.Name, hashSum)
				err := json.Unmarshal([]byte(taskCache.TaskOutput), &run.Status.Results)
				if err != nil {
					logger.Errorf("error while unmarshal of task output. %v", err)
				}
				run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonCacheHit.String(),
					"A cached result of the previous run was found.")
				return nil
			}
		}
		if err != nil {
			logger.Warnf("failed marshalling the spec, for pipelineloop: %s", pipelineLoopMeta.Name)
		}
	}
	if highestIteration > 0 {
		updateRunStatus(run, "last-idx", fmt.Sprintf("%d", highestIteration))
		updateRunStatus(run, "last-elem", fmt.Sprintf("%s", iterationElements[highestIteration-1]))
	}
	// Run is cancelled, just cancel all the running instance and return
	if run.IsCancelled() {
		if len(failedPrs) > 0 {
			run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailed.String(),
				"Run %s/%s was failed",
				run.Namespace, run.Name)
		} else {
			reason := pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String()
			if run.HasTimedOut(c.clock) { // This check is only possible if we are on tekton 0.27.0 +
				reason = v1alpha1.RunReasonTimedOut
			}
			run.Status.MarkRunFailed(reason, "Run %s/%s was cancelled", run.Namespace, run.Name)
		}

		for _, currentRunningPr := range currentRunningPrs {
			logger.Infof("Run %s/%s is cancelled.  Cancelling PipelineRun %s.", run.Namespace, run.Name, currentRunningPr.Name)
			if _, err := c.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Patch(ctx, currentRunningPr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntCancel.String(),
					"Failed to patch PipelineRun `%s` with cancellation: %v", currentRunningPr.Name, err)
				return nil
			}
		}
		return nil
	}

	// Run may be marked succeeded already by updatePipelineRunStatus
	if run.IsSuccessful() {
		return nil
	}

	retriesDone := len(run.Status.RetriesStatus)
	retries := run.Spec.Retries
	if retriesDone < retries && failedPrs != nil && len(failedPrs) > 0 {
		logger.Infof("RetriesDone: %d, Total Retries: %d", retriesDone, retries)
		run.Status.RetriesStatus = append(run.Status.RetriesStatus, v1alpha1.RunStatus{
			Status: duckv1.Status{
				ObservedGeneration: 0,
				Conditions:         run.Status.Conditions.DeepCopy(),
				Annotations:        nil,
			},
			RunStatusFields: runv1alpha1.RunStatusFields{
				StartTime:      run.Status.StartTime.DeepCopy(),
				CompletionTime: run.Status.CompletionTime.DeepCopy(),
				Results:        run.Status.Results,
				RetriesStatus:  nil,
				ExtraFields:    runtime.RawExtension{},
			},
		})
		// Without immediately updating here, wrong number of retries are performed.
		_, err := c.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).UpdateStatus(ctx, run, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		for _, failedPr := range failedPrs {
			// PipelineRun do not support a retry, we dispose off old PR and create a fresh one.
			// instead of deleting we just label it deleted=True.
			deletedLabel := map[string]string{"deleted": "True"}
			mergePatch := map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": deletedLabel,
				},
			}
			patch, err := json.Marshal(mergePatch)
			if err != nil {
				return err
			}
			_, _ = c.pipelineClientSet.TektonV1beta1().PipelineRuns(failedPr.Namespace).
				Patch(ctx, failedPr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			pr, err := c.createPipelineRun(ctx, logger, pipelineLoopSpec, run, highestIteration, iterationElements)
			if err != nil {
				return fmt.Errorf("error creating PipelineRun from Run %s while retrying: %w", run.Name, err)
			}
			status.PipelineRuns[pr.Name] = &pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{
				Iteration:     highestIteration,
				IterationItem: iterationElements[highestIteration-1],
				Status:        getPipelineRunStatusWithoutPipelineSpec(&pr.Status),
			}
			logger.Infof("Retried failed pipelineRun: %s with new pipelineRun: %s", failedPr.Name, pr.Name)
		}
		return nil
	}

	// Check the status of the PipelineRun for the highest iteration.
	if len(failedPrs) > 0 {
		for _, failedPr := range failedPrs {
			if status.CurrentRunning == 0 {
				run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailed.String(),
					"PipelineRun %s has failed", failedPr.Name)
			} else {
				run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
					"PipelineRun %s has failed", failedPr.Name)
			}
		}
		return nil
	}

	// Mark run status Running
	run.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
		"Iterations completed: %d", highestIteration-len(currentRunningPrs))

	// Move on to the next iteration (or the first iteration if there was no PipelineRun).
	// Check if the Run is done.
	nextIteration := highestIteration + 1
	if nextIteration > totalIterations {
		// Still running which we already marked, just waiting
		if len(currentRunningPrs) > 0 {
			logger.Infof("Already started all pipelineruns for the loop, totally %d pipelineruns, waiting for complete.", totalIterations)
			return nil
		}
		// All task finished
		run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
			"All PipelineRuns completed successfully")
		updateRunStatus(run, "condition", "succeeded")
		return nil
	}
	// Before starting up another PipelineRun, check if the run was cancelled.
	if run.IsCancelled() {
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String(),
			"Run %s/%s was cancelled",
			run.Namespace, run.Name)
		return nil
	}
	actualParallelism := 1
	// if Parallelism is bigger then totalIterations means there's no limit
	if pipelineLoopSpec.Parallelism > totalIterations {
		actualParallelism = totalIterations
	} else if pipelineLoopSpec.Parallelism > 0 {
		actualParallelism = pipelineLoopSpec.Parallelism
	}
	if len(currentRunningPrs) >= actualParallelism {
		logger.Infof("Currently %d pipelinerun started, meet parallelism %d, waiting...", len(currentRunningPrs), actualParallelism)
		return nil
	}

	// Create PipelineRun to run this iteration based on parallelism
	for i := 0; i < actualParallelism-len(currentRunningPrs); i++ {
		if isNestedPipelineLoop(pipelineLoopSpec) {
			maxNestedStackDepth, err := getMaxNestedStackDepth(pipelineLoopMeta)
			if err != nil {
				logger.Errorf("Error parsing max nested stack depth value: %v", err.Error())
				maxNestedStackDepth = DefaultNestedStackDepth
			}
			if maxNestedStackDepth > 0 {
				maxNestedStackDepth = maxNestedStackDepth - 1
				c.setMaxNestedStackDepth(ctx, pipelineLoopSpec, run, maxNestedStackDepth)
			} else if maxNestedStackDepth <= 0 {
				run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonStackLimitExceeded.String(), "nested stack depth limit reached.")
				return nil
			}
		}

		pr, err := c.createPipelineRun(ctx, logger, pipelineLoopSpec, run, nextIteration, iterationElements)
		if err != nil {
			return fmt.Errorf("error creating PipelineRun from Run %s: %w", run.Name, err)
		}
		status.PipelineRuns[pr.Name] = &pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{
			Iteration:     nextIteration,
			IterationItem: iterationElements[nextIteration-1],
			Status:        getPipelineRunStatusWithoutPipelineSpec(&pr.Status),
		}
		nextIteration++
		if nextIteration > totalIterations {
			logger.Infof("Started all pipelineruns for the loop, totally %d pipelineruns.", totalIterations)
			return nil
		}
	}

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
			return nil, nil, fmt.Errorf("error retrieving PipelineLoop for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		pipelineLoopMeta = tl.ObjectMeta
		pipelineLoopSpec = tl.Spec
	} else if run.Spec.Spec != nil {
		err := json.Unmarshal(run.Spec.Spec.Spec.Raw, &pipelineLoopSpec)
		if err != nil {
			run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
				"Error unmarshal PipelineLoop spec for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("error unmarshal PipelineLoop spec for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		pipelineLoopMeta = metav1.ObjectMeta{Name: run.Name,
			Namespace:       run.Namespace,
			OwnerReferences: run.OwnerReferences,
			Labels:          run.Spec.Spec.Metadata.Labels,
			Annotations:     run.Spec.Spec.Metadata.Annotations}
	} else {
		// Run does not require name but for PipelineLoop it does.
		run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
			"Missing spec.ref.name for Run %s/%s",
			run.Namespace, run.Name)
		return nil, nil, fmt.Errorf("missing spec.ref.name for Run %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name))
	}
	// pass down the run's serviceAccountName and podTemplate if they were not configured in the loop spec
	if pipelineLoopSpec.PodTemplate == nil && run.Spec.PodTemplate != nil {
		pipelineLoopSpec.PodTemplate = run.Spec.PodTemplate
	}
	if pipelineLoopSpec.ServiceAccountName == "" && run.Spec.ServiceAccountName != "" && run.Spec.ServiceAccountName != "default" {
		pipelineLoopSpec.ServiceAccountName = run.Spec.ServiceAccountName
	}
	return &pipelineLoopMeta, &pipelineLoopSpec, nil
}

func updateRunStatus(run *v1alpha1.Run, resultName string, resultVal string) bool {
	indexResultLastIdx := -1
	// if Run already has resultName, then update it else append.
	for i, res := range run.Status.Results {
		if res.Name == resultName {
			indexResultLastIdx = i
		}
	}
	if indexResultLastIdx >= 0 {
		run.Status.Results[indexResultLastIdx] = v1alpha1.RunResult{
			Name:  resultName,
			Value: resultVal,
		}
	} else {
		run.Status.Results = append(run.Status.Results, v1alpha1.RunResult{
			Name:  resultName,
			Value: resultVal,
		})
	}
	return true
}

func (c *Reconciler) createPipelineRun(ctx context.Context, logger *zap.SugaredLogger, tls *pipelineloopv1alpha1.PipelineLoopSpec, run *v1alpha1.Run, iteration int, iterationElements []interface{}) (*v1beta1.PipelineRun, error) {

	// Create name for PipelineRun from Run name plus iteration number.
	prName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", run.Name, fmt.Sprintf("%05d", iteration)))
	pipelineRunAnnotations := getPipelineRunAnnotations(run)
	currentIndex := iteration - 1
	if currentIndex > len(iterationElements) {
		currentIndex = len(iterationElements) - 1
	}
	currentIterationItemBytes, _ := json.Marshal(iterationElements[currentIndex])
	pipelineRunAnnotations[pipelineloop.GroupName+pipelineLoopCurrentIterationItemAnnotationKey] = string(currentIterationItemBytes)

	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prName,
			Namespace: run.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(run,
				schema.GroupVersionKind{Group: "tekton.dev", Version: "v1alpha1", Kind: "Run"})},
			Labels:      getPipelineRunLabels(run, strconv.Itoa(iteration)),
			Annotations: pipelineRunAnnotations,
		},
		Spec: v1beta1.PipelineRunSpec{
			Params:             getParameters(run, tls, iteration, string(currentIterationItemBytes)),
			Timeout:            tls.Timeout,
			ServiceAccountName: tls.ServiceAccountName,
			PodTemplate:        tls.PodTemplate,
			Workspaces:         tls.Workspaces,
			TaskRunSpecs:       tls.TaskRunSpecs,
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

func (c *Reconciler) cancelAllPipelineRuns(ctx context.Context, run *v1alpha1.Run) error {
	logger := logging.FromContext(ctx)
	pipelineRunLabels := getPipelineRunLabels(run, "")
	currentRunningPrs, err := c.pipelineRunLister.PipelineRuns(run.Namespace).List(labels.SelectorFromSet(pipelineRunLabels))
	if err != nil {
		return fmt.Errorf("could not list PipelineRuns %#v", err)
	}
	for _, currentRunningPr := range currentRunningPrs {
		if !currentRunningPr.IsDone() && !currentRunningPr.IsCancelled() {
			logger.Infof("Cancelling PipelineRun %s.", currentRunningPr.Name)
			if _, err := c.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Patch(ctx, currentRunningPr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntCancel.String(),
					"Failed to patch PipelineRun `%s` with cancellation: %v", currentRunningPr.Name, err)
				return nil
			}
		}
	}
	return nil
}

func (c *Reconciler) updatePipelineRunStatus(ctx context.Context, iterationElements []interface{}, run *v1alpha1.Run, status *pipelineloopv1alpha1.PipelineLoopRunStatus) (int, []*v1beta1.PipelineRun, []*v1beta1.PipelineRun, error) {
	logger := logging.FromContext(ctx)
	highestIteration := 0
	var currentRunningPrs []*v1beta1.PipelineRun
	var failedPrs []*v1beta1.PipelineRun
	if status.PipelineRuns == nil {
		status.PipelineRuns = make(map[string]*pipelineloopv1alpha1.PipelineLoopPipelineRunStatus)
	}
	pipelineRunLabels := getPipelineRunLabels(run, "")
	pipelineRuns, err := c.pipelineRunLister.PipelineRuns(run.Namespace).List(labels.SelectorFromSet(pipelineRunLabels))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("could not list PipelineRuns %#v", err)
	}
	if len(pipelineRuns) == 0 {
		return 0, nil, nil, nil
	}
	status.CurrentRunning = 0
	for _, pr := range pipelineRuns {
		lbls := pr.GetLabels()
		if lbls["deleted"] == "True" {
			// PipelineRun is already retried, skipping...
			continue
		}
		iterationStr := lbls[pipelineloop.GroupName+pipelineLoopIterationLabelKey]
		iteration, err := strconv.Atoi(iterationStr)
		if err != nil {
			run.Status.MarkRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
				"Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			logger.Errorf("Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			return 0, nil, nil, nil
		}
		// when we just create pr in a forloop, the started time may be empty
		if !pr.IsDone() {
			status.CurrentRunning++
			currentRunningPrs = append(currentRunningPrs, pr)
		}
		if pr.IsDone() && !pr.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
			failedPrs = append(failedPrs, pr)
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
			skippedTaskList := pr.Status.SkippedTasks
			for _, task := range skippedTaskList {
				if task.Name == lastLoopTask {
					// Mark run successful and stop the loop pipelinerun
					run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(run, "condition", "pass")
				}
			}
		}
		status.PipelineRuns[pr.Name] = &pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{
			Iteration:     iteration,
			IterationItem: iterationElements[iteration-1],
			Status:        getPipelineRunStatusWithoutPipelineSpec(&pr.Status),
		}
		if iteration > highestIteration {
			highestIteration = iteration
		}
		if pr.Status.ChildReferences != nil {
			//fetch taskruns/runs status specifically for pipelineloop-break-operation first
			for _, child := range pr.Status.ChildReferences {
				if strings.HasPrefix(child.PipelineTaskName, "pipelineloop-break-operation") {
					switch child.Kind {
					case "TaskRun":
						tr, err := tkstatus.GetTaskRunStatusForPipelineTask(ctx, c.pipelineClientSet, run.Namespace, child)
						if err != nil {
							logger.Errorf("can not get status for TaskRun, %v", err)
							return 0, nil, nil, fmt.Errorf("could not get TaskRun %s."+
								" %#v", child.Name, err)
						}
						if pr.Status.TaskRuns == nil {
							pr.Status.TaskRuns = make(map[string]*v1beta1.PipelineRunTaskRunStatus)
						}
						pr.Status.TaskRuns[child.Name] = &v1beta1.PipelineRunTaskRunStatus{
							PipelineTaskName: child.PipelineTaskName,
							WhenExpressions:  child.WhenExpressions,
							Status:           tr.DeepCopy(),
						}
					case "Run":
						run, err := tkstatus.GetRunStatusForPipelineTask(ctx, c.pipelineClientSet, run.Namespace, child)
						if err != nil {
							logger.Errorf("can not get status for Run, %v", err)
							return 0, nil, nil, fmt.Errorf("could not get Run %s."+
								" %#v", child.Name, err)
						}
						if pr.Status.Runs == nil {
							pr.Status.Runs = make(map[string]*v1beta1.PipelineRunRunStatus)
						}

						pr.Status.Runs[child.Name] = &v1beta1.PipelineRunRunStatus{
							PipelineTaskName: child.PipelineTaskName,
							WhenExpressions:  child.WhenExpressions,
							Status:           run.DeepCopy(),
						}
					default:
						//ignore
					}
				}
			}
		}
		for _, runStatus := range pr.Status.Runs {
			if strings.HasPrefix(runStatus.PipelineTaskName, "pipelineloop-break-operation") {
				if runStatus.Status != nil && !runStatus.Status.GetCondition(apis.ConditionSucceeded).IsUnknown() {
					err = c.cancelAllPipelineRuns(ctx, run)
					if err != nil {
						return 0, nil, nil, fmt.Errorf("could not cancel PipelineRuns belonging to Run %s."+
							" %#v", run.Name, err)
					}
					// Mark run successful and stop the loop pipelinerun
					run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(run, "condition", "pass")
					break
				}
			}
		}
		for _, taskRunStatus := range pr.Status.TaskRuns {
			if strings.HasPrefix(taskRunStatus.PipelineTaskName, "pipelineloop-break-operation") {
				if !taskRunStatus.Status.GetCondition(apis.ConditionSucceeded).IsUnknown() {
					err = c.cancelAllPipelineRuns(ctx, run)
					if err != nil {
						return 0, nil, nil, fmt.Errorf("could not cancel PipelineRuns belonging to task run %s."+
							" %#v", run.Name, err)
					}
					// Mark run successful and stop the loop pipelinerun
					run.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(run, "condition", "pass")
					break
				}
			}
		}
	}
	return highestIteration, currentRunningPrs, failedPrs, nil
}

func getIntegerParamValue(parm v1beta1.Param) (int, error) {
	fromStr := strings.TrimSuffix(parm.Value.StringVal, "\n")
	fromStr = strings.Trim(fromStr, " ")
	retVal, err := strconv.Atoi(fromStr)
	if err != nil {
		err = fmt.Errorf("input \"%s\" is not a number", parm.Name)
	}
	return retVal, err
}

func computeIterations(run *v1alpha1.Run, tls *pipelineloopv1alpha1.PipelineLoopSpec) (int, []interface{}, error) {
	// Find the iterate parameter.
	numberOfIterations := -1
	from := 0
	step := 1
	to := 0
	fromProvided := false
	toProvided := false
	iterationElements := []interface{}{}
	iterationParamStr := ""
	iterationParamStrSeparator := ""
	var err error
	for _, p := range run.Spec.Params {
		if p.Name == "from" {
			from, err = getIntegerParamValue(p)
			if err == nil {
				fromProvided = true
			}
		}
		if p.Name == "step" {
			step, err = getIntegerParamValue(p)
			if err != nil {
				return 0, iterationElements, err
			}
		}
		if p.Name == "to" {
			to, err = getIntegerParamValue(p)
			if err == nil {
				toProvided = true
			}
		}
		if p.Name == tls.IterateParam {
			if p.Value.Type == v1beta1.ParamTypeString {
				iterationParamStr = p.Value.StringVal
			}
			if p.Value.Type == v1beta1.ParamTypeArray {
				numberOfIterations = len(p.Value.ArrayVal)
				for _, v := range p.Value.ArrayVal {
					iterationElements = append(iterationElements, v)
				}
				break
			}
		}
		if p.Name == tls.IterateParamSeparator {
			iterationParamStrSeparator = p.Value.StringVal
		}

	}
	if iterationParamStr != "" {
		// Transfer p.Value to Array.
		err = nil //reset the err
		if iterationParamStrSeparator != "" {
			stringArr := strings.Split(iterationParamStr, iterationParamStrSeparator)
			numberOfIterations = len(stringArr)
			for _, v := range stringArr {
				iterationElements = append(iterationElements, v)
			}
		} else {
			var stringArr []string
			var ints []int
			var dictsString []map[string]string
			var dictsInt []map[string]int
			errString := json.Unmarshal([]byte(iterationParamStr), &stringArr)
			errInt := json.Unmarshal([]byte(iterationParamStr), &ints)
			errDictString := json.Unmarshal([]byte(iterationParamStr), &dictsString)
			errDictInt := json.Unmarshal([]byte(iterationParamStr), &dictsInt)
			if errString != nil && errInt != nil && errDictString != nil && errDictInt != nil {
				//try the default separator comma (,) in last
				if strings.Contains(iterationParamStr, defaultIterationParamStrSeparator) {
					stringArr := strings.Split(iterationParamStr, defaultIterationParamStrSeparator)
					numberOfIterations = len(stringArr)
					for _, v := range stringArr {
						iterationElements = append(iterationElements, v)
					}
				} else {
					return 0, iterationElements, fmt.Errorf("the value of the iterate parameter %q can not transfer to array", tls.IterateParam)
				}
			}
			if errString == nil {
				numberOfIterations = len(stringArr)
				for _, v := range stringArr {
					iterationElements = append(iterationElements, v)
				}
			} else if errInt == nil {
				numberOfIterations = len(ints)
				for _, v := range ints {
					iterationElements = append(iterationElements, v)
				}
			} else if errDictString == nil {
				numberOfIterations = len(dictsString)
				for _, v := range dictsString {
					iterationElements = append(iterationElements, v)
				}
			} else if errDictInt == nil {
				numberOfIterations = len(dictsInt)
				for _, v := range dictsInt {
					iterationElements = append(iterationElements, v)
				}
			}
		}
	}
	if from != to && fromProvided && toProvided {
		if step == 0 {
			return 0, iterationElements, fmt.Errorf("invalid values step: %d found in runs", step)
		}
		if (to-from < step && step > 0) || (to-from > step && step < 0) {
			// This is a special case, to emulate "python's enumerate" behaviour see issue #935
			numberOfIterations = 1
			iterationElements = append(iterationElements, from)
			return numberOfIterations, iterationElements, nil
		}
		if (from > to && step > 0) || (from < to && step < 0) {
			return 0, iterationElements, fmt.Errorf("invalid values for from:%d, to:%d & step: %d found in runs", from, to, step)
		}
		numberOfIterations = 0
		if step < 0 && from > to {
			for i := from; i >= to; i = i + step {
				numberOfIterations = numberOfIterations + 1
				iterationElements = append(iterationElements, i)
			}
		} else {
			for i := from; i <= to; i = i + step {
				numberOfIterations = numberOfIterations + 1
				iterationElements = append(iterationElements, i)
			}
		}
	}
	if from == to && step != 0 && fromProvided && toProvided {
		// This is a special case, to emulate "python's enumerate" behaviour see issue #935
		numberOfIterations = 1
		iterationElements = append(iterationElements, from)
	}
	return numberOfIterations, iterationElements, err
}

func getParameters(run *v1alpha1.Run, tls *pipelineloopv1alpha1.PipelineLoopSpec, iteration int, currentIterationItem string) []v1beta1.Param {
	var out []v1beta1.Param
	if tls.IterateParam != "" {
		// IterateParam defined
		var iterationParam, iterationParamStrSeparator *v1beta1.Param
		var item, separator v1beta1.Param
		for i, p := range run.Spec.Params {
			if p.Name == tls.IterateParam {
				if p.Value.Type == v1beta1.ParamTypeArray {
					out = append(out, v1beta1.Param{
						Name:  p.Name,
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: p.Value.ArrayVal[iteration-1]},
					})
				}
				if p.Value.Type == v1beta1.ParamTypeString {
					item = p
					iterationParam = &item
				}
			} else if p.Name == tls.IterateParamSeparator {
				separator = p
				iterationParamStrSeparator = &separator
			} else {
				out = append(out, run.Spec.Params[i])
			}
		}
		if iterationParam != nil {
			if iterationParamStrSeparator != nil && iterationParamStrSeparator.Value.StringVal != "" {
				iterationParamStr := iterationParam.Value.StringVal
				stringArr := strings.Split(iterationParamStr, iterationParamStrSeparator.Value.StringVal)
				out = append(out, v1beta1.Param{
					Name:  iterationParam.Name,
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: stringArr[iteration-1]},
				})

			} else {
				var stringArr []string
				var ints []int
				var dictsString []map[string]string
				var dictsInt []map[string]int
				iterationParamStr := iterationParam.Value.StringVal
				errString := json.Unmarshal([]byte(iterationParamStr), &stringArr)
				errInt := json.Unmarshal([]byte(iterationParamStr), &ints)
				errDictString := json.Unmarshal([]byte(iterationParamStr), &dictsString)
				errDictInt := json.Unmarshal([]byte(iterationParamStr), &dictsInt)
				if errString == nil {
					out = append(out, v1beta1.Param{
						Name:  iterationParam.Name,
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: stringArr[iteration-1]},
					})
				} else if errInt == nil {
					out = append(out, v1beta1.Param{
						Name:  iterationParam.Name,
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: strconv.Itoa(ints[iteration-1])},
					})
				} else if errDictString == nil {
					for dictParam := range dictsString[iteration-1] {
						out = append(out, v1beta1.Param{
							Name:  iterationParam.Name + "-subvar-" + dictParam,
							Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: dictsString[iteration-1][dictParam]},
						})
					}
				} else if errDictInt == nil {
					for dictParam := range dictsInt[iteration-1] {
						out = append(out, v1beta1.Param{
							Name:  iterationParam.Name + "-subvar-" + dictParam,
							Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: strconv.Itoa(dictsInt[iteration-1][dictParam])},
						})
					}
				} else {
					//try the default separator ","
					if strings.Contains(iterationParamStr, defaultIterationParamStrSeparator) {
						stringArr := strings.Split(iterationParamStr, defaultIterationParamStrSeparator)
						out = append(out, v1beta1.Param{
							Name:  iterationParam.Name,
							Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: stringArr[iteration-1]},
						})
					}
				}
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
	}
	if tls.IterationNumberParam != "" {
		out = append(out, v1beta1.Param{
			Name:  tls.IterationNumberParam,
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: strconv.Itoa(iteration)},
		})
	}
	if tls.IterateNumeric != "" {
		out = append(out, v1beta1.Param{
			Name:  tls.IterateNumeric,
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: currentIterationItem},
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
	ignoreLabelsKey := []string{"tekton.dev/pipelineRun", "tekton.dev/pipelineTask", "tekton.dev/pipeline", "custom.tekton.dev/pipelineLoopIteration"}
	for key, val := range run.ObjectMeta.Labels {
		if _, found := Find(ignoreLabelsKey, key); !found {
			labels[key] = val
		}
	}
	// Note: The Run label uses the normal Tekton group name.
	labels[pipeline.GroupName+pipelineLoopRunLabelKey] = run.Name
	if iterationStr != "" {
		labels[pipelineloop.GroupName+pipelineLoopIterationLabelKey] = iterationStr
	}
	labels[pipelineloop.GroupName+parentPRKey] = run.ObjectMeta.Labels["tekton.dev/pipelineRun"]

	var prOriginalName string
	if _, ok := run.ObjectMeta.Labels[pipelineloop.GroupName+originalPRKey]; ok {
		prOriginalName = run.ObjectMeta.Labels[pipelineloop.GroupName+originalPRKey]
	} else {
		prOriginalName = run.ObjectMeta.Labels["tekton.dev/pipelineRun"]
	}
	labels[pipelineloop.GroupName+originalPRKey] = prOriginalName
	// Empty the RunId reference from the KFP persistent agent because LabelKeyWorkflowRunId should be unique across all pipelineruns
	_, ok := labels[LabelKeyWorkflowRunId]
	if ok {
		delete(labels, LabelKeyWorkflowRunId)
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

// Storing PipelineSpec and TaskSpec in PipelineRunStatus is a source of significant memory consumption and OOM failures.
// Additionally, performance of status update in Run reconciler is impacted.
// PipelineSpec and TaskSpec seems to be redundant in this place.
// See issue: https://github.com/kubeflow/kfp-tekton/issues/962
func getPipelineRunStatusWithoutPipelineSpec(status *v1beta1.PipelineRunStatus) *v1beta1.PipelineRunStatus {
	s := status.DeepCopy()
	s.PipelineSpec = nil
	if s.TaskRuns != nil {
		for _, taskRun := range s.TaskRuns {
			taskRun.Status.TaskSpec = nil
		}
	}
	return s
}
