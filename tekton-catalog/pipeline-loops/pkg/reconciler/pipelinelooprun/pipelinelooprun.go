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
	kfptask "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/common"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	customRunReconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1"
	listersV1beta1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
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
	customRunLister       listersV1beta1.CustomRunLister
	pipelineLoopLister    listerspipelineloop.PipelineLoopLister
	pipelineRunLister     listers.PipelineRunLister
	cacheStore            *cache.TaskCacheStore
	clock                 clock.RealClock
	runKFPV2Driver        string
}
type CacheKey struct {
	Params           []tektonv1.Param                       `json:"params"`
	PipelineLoopSpec *pipelineloopv1alpha1.PipelineLoopSpec `json:"pipelineSpec"`
}

var (
	// Check that our Reconciler implements runreconciler.Interface
	_                customRunReconciler.Interface = (*Reconciler)(nil)
	cancelPatchBytes []byte
	iterationLimit   int = DefaultIterationLimit
)

func init() {
	var err error
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     tektonv1.PipelineRunSpecStatusCancelled,
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

func isCachingEnabled(run *tektonv1beta1.CustomRun) bool {
	return run.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] == "true"
}

func paramConvertTo(ctx context.Context, p *tektonv1beta1.Param, sink *tektonv1.Param) {
	sink.Name = p.Name
	newValue := tektonv1.ParamValue{}
	if p.Value.Type != "" {
		newValue.Type = tektonv1.ParamType(p.Value.Type)
	} else {
		newValue.Type = tektonv1.ParamType(v1beta1.ParamTypeString)
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

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the CustomRun resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, customRun *tektonv1beta1.CustomRun) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling CustomRun %s/%s at %v", customRun.Namespace, customRun.Name, time.Now())
	if customRun.Spec.CustomRef != nil && customRun.Spec.CustomSpec != nil {
		logger.Errorf("CustomRun %s/%s can provide one of CustomRun.Spec.CustomRef/CustomRun.Spec.CustomSpec", customRun.Namespace, customRun.Name)
		return nil
	}
	if customRun.Spec.CustomSpec == nil && customRun.Spec.CustomRef == nil {
		logger.Errorf("CustomRun %s/%s does not provide a spec or ref.", customRun.Namespace, customRun.Name)
		return nil
	}
	if (customRun.Spec.CustomRef != nil && customRun.Spec.CustomRef.Kind == pipelineloop.BreakTaskName) ||
		(customRun.Spec.CustomSpec != nil && customRun.Spec.CustomSpec.Kind == pipelineloop.BreakTaskName) {
		if !customRun.IsDone() {
			customRun.Status.InitializeConditions()
			customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
				"Break task is a dummy task.")
		}
		logger.Infof("Break task encountered %s", customRun.Name)
		return nil
	}
	// Check that the CustomRun references a PipelineLoop CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	if customRun.Spec.CustomRef != nil &&
		(customRun.Spec.CustomRef.APIVersion != pipelineloopv1alpha1.SchemeGroupVersion.String() ||
			customRun.Spec.CustomRef.Kind != pipelineloop.PipelineLoopControllerName) {
		logger.Errorf("Received control for a CustomRun %s/%s/%v that does not reference a PipelineLoop custom CRD ref", customRun.Namespace, customRun.Name, customRun.Spec.CustomRef)
		return nil
	}

	if customRun.Spec.CustomSpec != nil &&
		(customRun.Spec.CustomSpec.APIVersion != pipelineloopv1alpha1.SchemeGroupVersion.String() ||
			customRun.Spec.CustomSpec.Kind != pipelineloop.PipelineLoopControllerName) {
		logger.Errorf("Received control for a CustomRun %s/%s that does not reference a PipelineLoop custom CRD spec", customRun.Namespace, customRun.Name)
		return nil
	}
	logger.Infof("Received control for a CustomRun %s/%s %-v", customRun.Namespace, customRun.Name, customRun.Spec.CustomSpec)
	// If the CustomRun has not started, initialize the Condition and set the start time.
	firstIteration := false
	if !customRun.HasStarted() {
		logger.Infof("Starting new CustomRun %s/%s", customRun.Namespace, customRun.Name)
		customRun.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if customRun.Status.StartTime.Sub(customRun.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s createTimestamp %s is after the CustomRun started %s", customRun.Name, customRun.CreationTimestamp, customRun.Status.StartTime)
			customRun.Status.StartTime = &customRun.CreationTimestamp
		}
		// Emit events. During the first reconcile the status of the CustomRun may change twice
		// from not Started to Started and then to Running, so we need to sent the event here
		// and at the end of 'Reconcile' again.
		// We also want to send the "Started" event as soon as possible for anyone who may be waiting
		// on the event to perform user facing initialisations, such has reset a CI check status
		afterCondition := customRun.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, customRun)
		firstIteration = true
	}

	// Store the condition before reconcile
	beforeCondition := customRun.Status.GetCondition(apis.ConditionSucceeded)

	status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
	if err := customRun.Status.DecodeExtraFields(status); err != nil {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	if customRun.IsDone() {
		if customRun.IsSuccessful() && !c.cacheStore.Disabled && isCachingEnabled(customRun) {
			marshal, err := json.Marshal(CacheKey{
				PipelineLoopSpec: status.PipelineLoopSpec,
				Params:           v1ParamsConversion(ctx, customRun.Spec.Params),
			})
			if err == nil {
				hashSum := fmt.Sprintf("%x", md5.Sum(marshal))
				resultBytes, err1 := json.Marshal(customRun.Status.Results)
				if err1 != nil {
					return fmt.Errorf("error while marshalling result to cache for CustomRun: %s, %w", customRun.Name, err)
				}
				err = c.cacheStore.Put(&model.TaskCache{
					TaskHashKey: hashSum,
					TaskOutput:  string(resultBytes),
				})
				if err != nil {
					return fmt.Errorf("error while adding result to cache for CustomRun: %s, %w", customRun.Name, err)
				}
				logger.Infof("cached the results of successful CustomRun %s, with key: %s", customRun.Name, hashSum)
			}
		}
		logger.Infof("CustomRun %s/%s is done", customRun.Namespace, customRun.Name)
		return nil
	}
	// Reconcile the Run
	if err := c.reconcile(ctx, customRun, status, firstIteration); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
	}

	if err := c.updateLabelsAndAnnotations(ctx, customRun); err != nil {
		logger.Warn("Failed to update CustomRun labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	if err := customRun.Status.EncodeExtraFields(status); err != nil {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonInternalError.String(),
			"Internal error calling EncodeExtraFields: %v", err)
		logger.Errorf("EncodeExtraFields error: %v", err.Error())
	}

	afterCondition := customRun.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, customRun)
	return merr
}

func EnableCustomTaskFeatureFlag(ctx context.Context) context.Context {
	defaults, _ := config.NewDefaultsFromMap(map[string]string{})
	featureFlags, _ := config.NewFeatureFlagsFromMap(map[string]string{})
	c := &config.Config{
		Defaults:     defaults,
		FeatureFlags: featureFlags,
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

func (c *Reconciler) setMaxNestedStackDepth(ctx context.Context, pipelineLoopSpec *pipelineloopv1alpha1.PipelineLoopSpec, customRun *tektonv1beta1.CustomRun, depth int) {
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
				tl, err := c.pipelineLoopLister.PipelineLoops(customRun.Namespace).Get(t.TaskRef.Name)
				if err == nil && tl != nil {
					if len(tl.ObjectMeta.Annotations) == 0 {
						tl.ObjectMeta.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
					} else {
						tl.ObjectMeta.Annotations[MaxNestedStackDepthKey] = fmt.Sprint(depth)
					}
					_, err := c.pipelineloopClientSet.CustomV1alpha1().PipelineLoops(customRun.Namespace).Update(ctx, tl, metav1.UpdateOptions{})
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

func (c *Reconciler) reconcile(ctx context.Context, customRun *tektonv1beta1.CustomRun, status *pipelineloopv1alpha1.PipelineLoopRunStatus, firstIteration bool) error {
	ctx = EnableCustomTaskFeatureFlag(ctx)
	logger := logging.FromContext(ctx)
	var hashSum string
	// Get the PipelineLoop referenced by the CustomRun
	pipelineLoopMeta, pipelineLoopSpec, err := c.getPipelineLoop(ctx, customRun, firstIteration)
	if err != nil {
		return nil
	}
	// Store the fetched PipelineLoopSpec on the CustomRun for auditing
	storePipelineLoopSpec(status, pipelineLoopSpec)

	// Propagate labels and annotations from PipelineLoop to Run.
	propagatePipelineLoopLabelsAndAnnotations(customRun, pipelineLoopMeta)

	pipelineLoopSpec.SetDefaults(ctx)
	// Validate PipelineLoop spec
	if err := pipelineLoopSpec.Validate(ctx); err != nil {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"PipelineLoop %s/%s can't be CustomRun; it has an invalid spec: %s",
			pipelineLoopMeta.Namespace, pipelineLoopMeta.Name, err)
		return nil
	}

	// Determine how many iterations of the Task will be done.
	totalIterations, iterationElements, err := computeIterations(customRun, pipelineLoopSpec)
	if err != nil {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"Cannot determine number of iterations: %s", err)
		return nil
	}
	if totalIterations > iterationLimit {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
			"Total number of iterations exceeds the limit: %d", iterationLimit)
		return nil
	}

	// Update status of PipelineRuns.  Return the PipelineRun representing the highest loop iteration.
	highestIteration, currentRunningPrs, failedPrs, err := c.updatePipelineRunStatus(ctx, iterationElements, customRun, status)
	if err != nil {
		return fmt.Errorf("error updating PipelineRun status for CustomRun %s/%s: %w", customRun.Namespace, customRun.Name, err)
	}
	if !c.cacheStore.Disabled && isCachingEnabled(customRun) {
		marshal, err := json.Marshal(CacheKey{
			PipelineLoopSpec: pipelineLoopSpec,
			Params:           v1ParamsConversion(ctx, customRun.Spec.Params),
		})
		if marshal != nil && err == nil {
			hashSum = fmt.Sprintf("%x", md5.Sum(marshal))
			taskCache, err := c.cacheStore.Get(hashSum)
			if err == nil && taskCache != nil {
				logger.Infof("Found a cached entry, for customRun: %s, with key:", customRun.Name, hashSum)
				err := json.Unmarshal([]byte(taskCache.TaskOutput), &customRun.Status.Results)
				if err != nil {
					logger.Errorf("error while unmarshal of task output. %v", err)
				}
				customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonCacheHit.String(),
					"A cached result of the previous run was found.")
				return nil
			}
		}
		if err != nil {
			logger.Warnf("failed marshalling the spec, for pipelineloop: %s", pipelineLoopMeta.Name)
		}
	}
	if highestIteration > 0 {
		updateRunStatus(customRun, "last-idx", fmt.Sprintf("%d", highestIteration))
		updateRunStatus(customRun, "last-elem", fmt.Sprintf("%s", iterationElements[highestIteration-1]))
	}
	// CustomRun is cancelled, just cancel all the running instance and return
	if customRun.IsCancelled() {
		if len(failedPrs) > 0 {
			customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailed.String(),
				"CustomRun %s/%s was failed",
				customRun.Namespace, customRun.Name)
		} else {
			reason := pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String()
			if customRun.HasTimedOut(c.clock) { // This check is only possible if we are on tekton 0.27.0 +
				reason = string(tektonv1beta1.CustomRunReasonTimedOut)
			}
			customRun.Status.MarkCustomRunFailed(reason, "CustomRun %s/%s was cancelled", customRun.Namespace, customRun.Name)
		}

		for _, currentRunningPr := range currentRunningPrs {
			logger.Infof("CustomRun %s/%s is cancelled.  Cancelling PipelineRun %s.", customRun.Namespace, customRun.Name, currentRunningPr.Name)
			if _, err := c.pipelineClientSet.TektonV1().PipelineRuns(customRun.Namespace).Patch(ctx, currentRunningPr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				customRun.Status.MarkCustomRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntCancel.String(),
					"Failed to patch PipelineRun `%s` with cancellation: %v", currentRunningPr.Name, err)
				return nil
			}
		}
		return nil
	}

	// CustomRun may be marked succeeded already by updatePipelineRunStatus
	if customRun.IsSuccessful() {
		return nil
	}

	retriesDone := len(customRun.Status.RetriesStatus)
	retries := customRun.Spec.Retries
	if retriesDone < retries && failedPrs != nil && len(failedPrs) > 0 {
		logger.Infof("RetriesDone: %d, Total Retries: %d", retriesDone, retries)
		customRun.Status.RetriesStatus = append(customRun.Status.RetriesStatus, tektonv1beta1.CustomRunStatus{
			Status: duckv1.Status{
				ObservedGeneration: 0,
				Conditions:         customRun.Status.Conditions.DeepCopy(),
				Annotations:        nil,
			},
			CustomRunStatusFields: tektonv1beta1.CustomRunStatusFields{
				StartTime:      customRun.Status.StartTime.DeepCopy(),
				CompletionTime: customRun.Status.CompletionTime.DeepCopy(),
				Results:        customRun.Status.Results,
				RetriesStatus:  nil,
				ExtraFields:    runtime.RawExtension{},
			},
		})
		// Without immediately updating here, wrong number of retries are performed.
		_, err := c.pipelineClientSet.TektonV1beta1().CustomRuns(customRun.Namespace).UpdateStatus(ctx, customRun, metav1.UpdateOptions{})
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
			_, _ = c.pipelineClientSet.TektonV1().PipelineRuns(failedPr.Namespace).
				Patch(ctx, failedPr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			pr, err := c.createPipelineRun(ctx, logger, pipelineLoopSpec, customRun, highestIteration, iterationElements)
			if err != nil {
				return fmt.Errorf("error creating PipelineRun from CustomRun %s while retrying: %w", customRun.Name, err)
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
				customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailed.String(),
					"PipelineRun %s has failed", failedPr.Name)
			} else {
				customRun.Status.MarkCustomRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
					"PipelineRun %s has failed", failedPr.Name)
			}
		}
		return nil
	}

	// Mark customRun status Running
	customRun.Status.MarkCustomRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(),
		"Iterations completed: %d", highestIteration-len(currentRunningPrs))

	// Move on to the next iteration (or the first iteration if there was no PipelineRun).
	// Check if the CustomRun is done.
	nextIteration := highestIteration + 1
	if nextIteration > totalIterations {
		// Still running which we already marked, just waiting
		if len(currentRunningPrs) > 0 {
			logger.Infof("Already started all pipelineruns for the loop, totally %d pipelineruns, waiting for complete.", totalIterations)
			return nil
		}
		// All task finished
		customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
			"All PipelineRuns completed successfully")
		updateRunStatus(customRun, "condition", "succeeded")
		return nil
	}
	// Before starting up another PipelineRun, check if the customRun was cancelled.
	if customRun.IsCancelled() {
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCancelled.String(),
			"CustomRun %s/%s was cancelled",
			customRun.Namespace, customRun.Name)
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

	// Create PipelineRun to customRun this iteration based on parallelism
	for i := 0; i < actualParallelism-len(currentRunningPrs); i++ {
		if isNestedPipelineLoop(pipelineLoopSpec) {
			maxNestedStackDepth, err := getMaxNestedStackDepth(pipelineLoopMeta)
			if err != nil {
				logger.Errorf("Error parsing max nested stack depth value: %v", err.Error())
				maxNestedStackDepth = DefaultNestedStackDepth
			}
			if maxNestedStackDepth > 0 {
				maxNestedStackDepth = maxNestedStackDepth - 1
				c.setMaxNestedStackDepth(ctx, pipelineLoopSpec, customRun, maxNestedStackDepth)
			} else if maxNestedStackDepth <= 0 {
				customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonStackLimitExceeded.String(), "nested stack depth limit reached.")
				return nil
			}
		}

		pr, err := c.createPipelineRun(ctx, logger, pipelineLoopSpec, customRun, nextIteration, iterationElements)
		if err != nil {
			return fmt.Errorf("error creating PipelineRun from CustomRun %s: %w", customRun.Name, err)
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

func (c *Reconciler) getPipelineLoop(ctx context.Context, customRun *tektonv1beta1.CustomRun, firstIteration bool) (*metav1.ObjectMeta, *pipelineloopv1alpha1.PipelineLoopSpec, error) {
	pipelineLoopMeta := metav1.ObjectMeta{}
	pipelineLoopSpec := pipelineloopv1alpha1.PipelineLoopSpec{}
	if customRun.Spec.CustomRef != nil && customRun.Spec.CustomRef.Name != "" {
		// Use the k8 client to get the PipelineLoop rather than the Lister on the first reconcile. This avoids a timing issue where
		// the PipelineLoop is not yet in the lister cache if it is created at nearly the same time as the Run.
		// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
		var tl *pipelineloopv1alpha1.PipelineLoop
		var err error
		if firstIteration {
			tl, err = c.pipelineloopClientSet.CustomV1alpha1().PipelineLoops(customRun.Namespace).Get(ctx, customRun.Spec.CustomRef.Name, metav1.GetOptions{})
		} else {
			tl, err = c.pipelineLoopLister.PipelineLoops(customRun.Namespace).Get(customRun.Spec.CustomRef.Name)
		}
		if err != nil {
			customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
				"Error retrieving PipelineLoop for CustomRun %s/%s: %s",
				customRun.Namespace, customRun.Name, err)
			return nil, nil, fmt.Errorf("error retrieving PipelineLoop for CustomRun %s: %w", fmt.Sprintf("%s/%s", customRun.Namespace, customRun.Name), err)
		}
		pipelineLoopMeta = tl.ObjectMeta
		pipelineLoopSpec = tl.Spec
	} else if customRun.Spec.CustomSpec != nil {
		err := json.Unmarshal(customRun.Spec.CustomSpec.Spec.Raw, &pipelineLoopSpec)
		if err != nil {
			customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
				"Error unmarshal PipelineLoop spec for CustomRun %s/%s: %s",
				customRun.Namespace, customRun.Name, err)
			return nil, nil, fmt.Errorf("error unmarshal PipelineLoop spec for CustomRun %s: %w", fmt.Sprintf("%s/%s", customRun.Namespace, customRun.Name), err)
		}
		pipelineLoopMeta = metav1.ObjectMeta{Name: customRun.Name,
			Namespace:       customRun.Namespace,
			OwnerReferences: customRun.OwnerReferences,
			Labels:          customRun.Spec.CustomSpec.Metadata.Labels,
			Annotations:     customRun.Spec.CustomSpec.Metadata.Annotations}
	} else {
		// CustomRun does not require name but for PipelineLoop it does.
		customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop.String(),
			"Missing spec.customRef.name for CustomRun %s/%s",
			customRun.Namespace, customRun.Name)
		return nil, nil, fmt.Errorf("missing spec.customRef.name for CustomRun %s", fmt.Sprintf("%s/%s", customRun.Namespace, customRun.Name))
	}

	if pipelineLoopSpec.ServiceAccountName == "" && customRun.Spec.ServiceAccountName != "" && customRun.Spec.ServiceAccountName != "default" {
		pipelineLoopSpec.ServiceAccountName = customRun.Spec.ServiceAccountName
	}
	return &pipelineLoopMeta, &pipelineLoopSpec, nil
}

func updateRunStatus(customRun *tektonv1beta1.CustomRun, resultName string, resultVal string) bool {
	indexResultLastIdx := -1
	// if CustomRun already has resultName, then update it else append.
	for i, res := range customRun.Status.Results {
		if res.Name == resultName {
			indexResultLastIdx = i
		}
	}
	if indexResultLastIdx >= 0 {
		customRun.Status.Results[indexResultLastIdx] = tektonv1beta1.CustomRunResult{
			Name:  resultName,
			Value: resultVal,
		}
	} else {
		customRun.Status.Results = append(customRun.Status.Results, tektonv1beta1.CustomRunResult{
			Name:  resultName,
			Value: resultVal,
		})
	}
	return true
}

func (c *Reconciler) createPipelineRun(ctx context.Context, logger *zap.SugaredLogger, tls *pipelineloopv1alpha1.PipelineLoopSpec, customRun *tektonv1beta1.CustomRun, iteration int, iterationElements []interface{}) (*tektonv1.PipelineRun, error) {

	// Create name for PipelineRun from CustomRun name plus iteration number.
	prName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", customRun.Name, fmt.Sprintf("%05d", iteration)))
	pipelineRunAnnotations := getPipelineRunAnnotations(customRun)
	currentIndex := iteration - 1
	if currentIndex > len(iterationElements) {
		currentIndex = len(iterationElements) - 1
	}
	currentIterationItemBytes, _ := json.Marshal(iterationElements[currentIndex])
	pipelineRunAnnotations[pipelineloop.GroupName+pipelineLoopCurrentIterationItemAnnotationKey] = string(currentIterationItemBytes)
	pipelineRunParams := getParameters(customRun, tls, iteration, string(currentIterationItemBytes))
	// Run DAG Driver if KFP V2 is enabled
	if c.runKFPV2Driver == "true" {
		options, err := kfptask.ParseParams(customRun)
		if err != nil {
			logger.Errorf("Run %s/%s is invalid because of %s", customRun.Namespace, customRun.Name, err)
			customRun.Status.MarkCustomRunFailed(kfptask.ReasonFailedValidation,
				"Run can't be run because it has an invalid param - %v", err)
			return nil, err
		}
		tmp, _ := strconv.ParseInt(string(currentIterationItemBytes), 10, 32)
		kfptask.UpdateOptionsIterationIndex(options, int(tmp))

		_, _, executionID, _, _, driverErr := kfptask.ExecDriver(ctx, options)
		if driverErr != nil {
			logger.Errorf("kfp-driver execution failed when reconciling Run %s/%s: %v", customRun.Namespace, customRun.Name, driverErr)
			customRun.Status.MarkCustomRunFailed(kfptask.ReasonDriverError,
				"kfp-driver execution failed: %v", driverErr)
			return nil, err
		}
		for i, pipelineRunParam := range pipelineRunParams {
			if pipelineRunParam.Name == "dag-execution-id" {
				pipelineRunParam.Value.StringVal = executionID
				pipelineRunParams[i] = pipelineRunParam
				break
			}
		}
	}
	pr := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prName,
			Namespace: customRun.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(customRun,
				schema.GroupVersionKind{Group: "tekton.dev", Version: "v1beta1", Kind: "CustomRun"})},
			Labels:      getPipelineRunLabels(customRun, strconv.Itoa(iteration)),
			Annotations: pipelineRunAnnotations,
		},
		Spec: tektonv1.PipelineRunSpec{
			Params:   pipelineRunParams,
			Timeouts: nil,
			TaskRunTemplate: tektonv1.PipelineTaskRunTemplate{
				ServiceAccountName: tls.ServiceAccountName,
				PodTemplate:        tls.PodTemplate,
			},
			Workspaces:   tls.Workspaces,
			TaskRunSpecs: tls.TaskRunSpecs,
		}}
	if tls.Timeout != nil {
		pr.Spec.Timeouts = &tektonv1.TimeoutFields{Pipeline: tls.Timeout}
	}
	if tls.PipelineRef != nil {
		pr.Spec.PipelineRef = &tektonv1.PipelineRef{
			Name: tls.PipelineRef.Name,
			// Kind: tls.PipelineRef.Kind,
		}
	} else if tls.PipelineSpec != nil {
		pr.Spec.PipelineSpec = tls.PipelineSpec
	}

	logger.Infof("Creating a new PipelineRun object %s", prName)
	return c.pipelineClientSet.TektonV1().PipelineRuns(customRun.Namespace).Create(ctx, pr, metav1.CreateOptions{})

}

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, customRun *tektonv1beta1.CustomRun) error {
	newCustomRun, err := c.customRunLister.CustomRuns(customRun.Namespace).Get(customRun.Name)
	if err != nil {
		return fmt.Errorf("error getting CustomRun %s when updating labels/annotations: %w", customRun.Name, err)
	}
	if !reflect.DeepEqual(customRun.ObjectMeta.Labels, newCustomRun.ObjectMeta.Labels) || !reflect.DeepEqual(customRun.ObjectMeta.Annotations, newCustomRun.ObjectMeta.Annotations) {
		mergePatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels":      customRun.ObjectMeta.Labels,
				"annotations": customRun.ObjectMeta.Annotations,
			},
		}
		patch, err := json.Marshal(mergePatch)
		if err != nil {
			return err
		}
		_, err = c.pipelineClientSet.TektonV1beta1().CustomRuns(customRun.Namespace).Patch(ctx, customRun.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}

func (c *Reconciler) cancelAllPipelineRuns(ctx context.Context, customRun *tektonv1beta1.CustomRun) error {
	logger := logging.FromContext(ctx)
	pipelineRunLabels := getPipelineRunLabels(customRun, "")
	currentRunningPrs, err := c.pipelineRunLister.PipelineRuns(customRun.Namespace).List(labels.SelectorFromSet(pipelineRunLabels))
	if err != nil {
		return fmt.Errorf("could not list PipelineRuns %#v", err)
	}
	for _, currentRunningPr := range currentRunningPrs {
		if !currentRunningPr.IsDone() && !currentRunningPr.IsCancelled() {
			logger.Infof("Cancelling PipelineRun %s.", currentRunningPr.Name)
			if _, err := c.pipelineClientSet.TektonV1().PipelineRuns(customRun.Namespace).Patch(ctx, currentRunningPr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonCouldntCancel.String(),
					"Failed to patch PipelineRun `%s` with cancellation: %v", currentRunningPr.Name, err)
				return nil
			}
		}
	}
	return nil
}

func (c *Reconciler) updatePipelineRunStatus(ctx context.Context, iterationElements []interface{}, customRun *tektonv1beta1.CustomRun, status *pipelineloopv1alpha1.PipelineLoopRunStatus) (int, []*tektonv1.PipelineRun, []*tektonv1.PipelineRun, error) {
	logger := logging.FromContext(ctx)
	highestIteration := 0
	var currentRunningPrs []*tektonv1.PipelineRun
	var failedPrs []*tektonv1.PipelineRun
	if status.PipelineRuns == nil {
		status.PipelineRuns = make(map[string]*pipelineloopv1alpha1.PipelineLoopPipelineRunStatus)
	}
	pipelineRunLabels := getPipelineRunLabels(customRun, "")
	pipelineRuns, err := c.pipelineRunLister.PipelineRuns(customRun.Namespace).List(labels.SelectorFromSet(pipelineRunLabels))
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
			customRun.Status.MarkCustomRunFailed(pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation.String(),
				"Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			logger.Errorf("Error converting iteration number in PipelineRun %s:  %#v", pr.Name, err)
			return 0, nil, nil, nil
		}
		// when we just create pr in a forloop, the started time may be empty
		if !pr.IsDone() {
			status.CurrentRunning++
			currentRunningPrs = append(currentRunningPrs, pr)
		} else {
			var DAGStatus pb.Execution_State
			if !pr.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
				failedPrs = append(failedPrs, pr)
				DAGStatus = pb.Execution_FAILED
			} else {
				DAGStatus = pb.Execution_COMPLETE
			}
			// Run DAG Driver if KFP V2 is enabled and is not yet updated
			if c.runKFPV2Driver == "true" && lbls["updated"] != "True" {
				pipelineRunParams := pr.Spec.Params
				options, err := kfptask.ParseParams(customRun)
				if err != nil {
					logger.Errorf("Run %s/%s is invalid because of %s", customRun.Namespace, customRun.Name, err)
					customRun.Status.MarkCustomRunFailed(kfptask.ReasonFailedValidation,
						"Run can't be run because it has an invalid param - %v", err)
					return 0, nil, nil, err
				}
				var executionID string
				for _, pipelineRunParam := range pipelineRunParams {
					if pipelineRunParam.Name == "dag-execution-id" {
						executionID = pipelineRunParam.Value.StringVal
						break
					}
				}
				kfptask.UpdateOptionsDAGExecutionID(options, executionID)
				DAGErr := kfptask.UpdateDAGPublisher(ctx, options, DAGStatus)
				if err != nil {
					logger.Errorf("kfp publisher failed when reconciling Run %s/%s: %v", customRun.Namespace, customRun.Name, DAGErr)
					customRun.Status.MarkCustomRunFailed(kfptask.ReasonDriverError,
						"kfp publisher execution failed: %v", DAGErr)
					return 0, nil, nil, DAGErr
				}
				// add lable to skip updated pipelines
				updatedLabel := map[string]string{"updated": "True"}
				mergePatch := map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": updatedLabel,
					},
				}
				patch, updateErr := json.Marshal(mergePatch)
				if updateErr != nil {
					return 0, nil, nil, updateErr
				}
				_, _ = c.pipelineClientSet.TektonV1().PipelineRuns(pr.Namespace).
					Patch(ctx, pr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			}
		}

		// Mark customRun successful if the condition are met.
		// if the last loop task is skipped, but the highestIterationPr successed. Mark customRun success.
		// lastLoopTask := highestIterationPr.ObjectMeta.Annotations["last-loop-task"]
		lastLoopTask := ""
		for key, val := range customRun.ObjectMeta.Labels {
			if key == "last-loop-task" {
				lastLoopTask = val
			}
		}
		if lastLoopTask != "" {
			skippedTaskList := pr.Status.SkippedTasks
			for _, task := range skippedTaskList {
				if task.Name == lastLoopTask {
					// Mark customRun successful and stop the loop pipelinerun
					customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(customRun, "condition", "pass")
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
		taskrunstatuses := make(map[string]*tektonv1.PipelineRunTaskRunStatus)
		runstatuses := make(map[string]*tektonv1.PipelineRunRunStatus)
		if pr.Status.ChildReferences != nil {
			//fetch taskruns/runs status specifically for pipelineloop-break-operation first
			for _, child := range pr.Status.ChildReferences {
				if strings.HasPrefix(child.PipelineTaskName, "pipelineloop-break-operation") {
					switch child.Kind {
					case "TaskRun":
						tr, err := tkstatus.GetTaskRunStatusForPipelineTask(ctx, c.pipelineClientSet, customRun.Namespace, child)
						if err != nil {
							logger.Errorf("can not get status for TaskRun, %v", err)
							return 0, nil, nil, fmt.Errorf("could not get TaskRun %s."+
								" %#v", child.Name, err)
						}
						taskrunstatuses[child.Name] = &tektonv1.PipelineRunTaskRunStatus{
							PipelineTaskName: child.PipelineTaskName,
							WhenExpressions:  child.WhenExpressions,
							Status:           tr.DeepCopy(),
						}
					case "Run":
						run, err := tkstatus.GetCustomRunStatusForPipelineTask(ctx, c.pipelineClientSet, customRun.Namespace, child)
						if err != nil {
							logger.Errorf("can not get status for Run, %v", err)
							return 0, nil, nil, fmt.Errorf("could not get Run %s."+
								" %#v", child.Name, err)
						}
						runstatuses[child.Name] = &tektonv1.PipelineRunRunStatus{
							PipelineTaskName: child.PipelineTaskName,
							WhenExpressions:  child.WhenExpressions,
							Status:           run.DeepCopy(),
						}
					case "CustomRun":
						run, err := tkstatus.GetCustomRunStatusForPipelineTask(ctx, c.pipelineClientSet, customRun.Namespace, child)
						if err != nil {
							logger.Errorf("can not get status for CustomRun, %v", err)
							return 0, nil, nil, fmt.Errorf("could not get CustomRun %s."+
								" %#v", child.Name, err)
						}
						runstatuses[child.Name] = &tektonv1.PipelineRunRunStatus{
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
		for _, runStatus := range runstatuses {
			if strings.HasPrefix(runStatus.PipelineTaskName, "pipelineloop-break-operation") {
				if runStatus.Status != nil && !runStatus.Status.GetCondition(apis.ConditionSucceeded).IsUnknown() {
					err = c.cancelAllPipelineRuns(ctx, customRun)
					if err != nil {
						return 0, nil, nil, fmt.Errorf("could not cancel PipelineRuns belonging to customRun %s."+
							" %#v", customRun.Name, err)
					}
					// Mark customRun successful and stop the loop pipelinerun
					customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(customRun, "condition", "pass")
					break
				}
			}
		}
		for _, taskRunStatus := range taskrunstatuses {
			if strings.HasPrefix(taskRunStatus.PipelineTaskName, "pipelineloop-break-operation") {
				if !taskRunStatus.Status.GetCondition(apis.ConditionSucceeded).IsUnknown() {
					err = c.cancelAllPipelineRuns(ctx, customRun)
					if err != nil {
						return 0, nil, nil, fmt.Errorf("could not cancel PipelineRuns belonging to task customRun %s."+
							" %#v", customRun.Name, err)
					}
					// Mark customRun successful and stop the loop pipelinerun
					customRun.Status.MarkCustomRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(),
						"PipelineRuns completed successfully with the conditions are met")
					updateRunStatus(customRun, "condition", "pass")
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

func computeIterations(run *tektonv1beta1.CustomRun, tls *pipelineloopv1alpha1.PipelineLoopSpec) (int, []interface{}, error) {
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

func getParameters(customRun *tektonv1beta1.CustomRun, tls *pipelineloopv1alpha1.PipelineLoopSpec, iteration int, currentIterationItem string) []tektonv1.Param {
	var out []tektonv1.Param
	if tls.IterateParam != "" {
		// IterateParam defined
		var iterationParam, iterationParamStrSeparator *v1beta1.Param
		var item, separator v1beta1.Param
		for i, p := range customRun.Spec.Params {
			if p.Name == tls.IterateParam {
				if p.Value.Type == v1beta1.ParamTypeArray {
					out = append(out, tektonv1.Param{
						Name:  p.Name,
						Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: p.Value.ArrayVal[iteration-1]},
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
				v1Param := tektonv1.Param{}
				ctx := context.Background()
				paramConvertTo(ctx, &customRun.Spec.Params[i], &v1Param)
				out = append(out, v1Param)
			}
		}
		if iterationParam != nil {
			if iterationParamStrSeparator != nil && iterationParamStrSeparator.Value.StringVal != "" {
				iterationParamStr := iterationParam.Value.StringVal
				stringArr := strings.Split(iterationParamStr, iterationParamStrSeparator.Value.StringVal)
				out = append(out, tektonv1.Param{
					Name:  iterationParam.Name,
					Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: stringArr[iteration-1]},
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
					out = append(out, tektonv1.Param{
						Name:  iterationParam.Name,
						Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: stringArr[iteration-1]},
					})
				} else if errInt == nil {
					out = append(out, tektonv1.Param{
						Name:  iterationParam.Name,
						Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: strconv.Itoa(ints[iteration-1])},
					})
				} else if errDictString == nil {
					for dictParam := range dictsString[iteration-1] {
						out = append(out, tektonv1.Param{
							Name:  iterationParam.Name + "-subvar-" + dictParam,
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: dictsString[iteration-1][dictParam]},
						})
					}
				} else if errDictInt == nil {
					for dictParam := range dictsInt[iteration-1] {
						out = append(out, tektonv1.Param{
							Name:  iterationParam.Name + "-subvar-" + dictParam,
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: strconv.Itoa(dictsInt[iteration-1][dictParam])},
						})
					}
				} else {
					//try the default separator ","
					if strings.Contains(iterationParamStr, defaultIterationParamStrSeparator) {
						stringArr := strings.Split(iterationParamStr, defaultIterationParamStrSeparator)
						out = append(out, tektonv1.Param{
							Name:  iterationParam.Name,
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: stringArr[iteration-1]},
						})
					}
				}
			}
		}
	} else {
		// IterateNumeric defined
		IterateStrings := []string{"from", "step", "to"}
		for i, p := range customRun.Spec.Params {
			if _, found := Find(IterateStrings, p.Name); !found {
				v1Param := tektonv1.Param{}
				ctx := context.Background()
				paramConvertTo(ctx, &customRun.Spec.Params[i], &v1Param)
				out = append(out, v1Param)
			}
		}
	}
	if tls.IterationNumberParam != "" {
		out = append(out, tektonv1.Param{
			Name:  tls.IterationNumberParam,
			Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: strconv.Itoa(iteration)},
		})
	}
	if tls.IterateNumeric != "" {
		out = append(out, tektonv1.Param{
			Name:  tls.IterateNumeric,
			Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: currentIterationItem},
		})
	}
	return out
}

func getPipelineRunAnnotations(customRun *tektonv1beta1.CustomRun) map[string]string {
	// Propagate annotations from CustomRun to PipelineRun.
	annotations := make(map[string]string, len(customRun.ObjectMeta.Annotations)+1)
	for key, val := range customRun.ObjectMeta.Annotations {
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

func getPipelineRunLabels(customRun *tektonv1beta1.CustomRun, iterationStr string) map[string]string {
	// Propagate labels from CustomRun to PipelineRun.
	labels := make(map[string]string, len(customRun.ObjectMeta.Labels)+1)
	ignoreLabelsKey := []string{"tekton.dev/pipelineRun", "tekton.dev/pipelineTask", "tekton.dev/pipeline", "custom.tekton.dev/pipelineLoopIteration"}
	for key, val := range customRun.ObjectMeta.Labels {
		if _, found := Find(ignoreLabelsKey, key); !found {
			labels[key] = val
		}
	}
	// Note: The CustomRun label uses the normal Tekton group name.
	labels[pipeline.GroupName+pipelineLoopRunLabelKey] = customRun.Name
	if iterationStr != "" {
		labels[pipelineloop.GroupName+pipelineLoopIterationLabelKey] = iterationStr
	}
	labels[pipelineloop.GroupName+parentPRKey] = customRun.ObjectMeta.Labels["tekton.dev/pipelineRun"]

	var prOriginalName string
	if _, ok := customRun.ObjectMeta.Labels[pipelineloop.GroupName+originalPRKey]; ok {
		prOriginalName = customRun.ObjectMeta.Labels[pipelineloop.GroupName+originalPRKey]
	} else {
		prOriginalName = customRun.ObjectMeta.Labels["tekton.dev/pipelineRun"]
	}
	labels[pipelineloop.GroupName+originalPRKey] = prOriginalName
	// Empty the RunId reference from the KFP persistent agent because LabelKeyWorkflowRunId should be unique across all pipelineruns
	_, ok := labels[LabelKeyWorkflowRunId]
	if ok {
		delete(labels, LabelKeyWorkflowRunId)
	}
	return labels
}

func propagatePipelineLoopLabelsAndAnnotations(customRun *tektonv1beta1.CustomRun, pipelineLoopMeta *metav1.ObjectMeta) {
	// Propagate labels from PipelineLoop to customRun.
	if customRun.ObjectMeta.Labels == nil {
		customRun.ObjectMeta.Labels = make(map[string]string, len(pipelineLoopMeta.Labels)+1)
	}
	for key, value := range pipelineLoopMeta.Labels {
		customRun.ObjectMeta.Labels[key] = value
	}
	customRun.ObjectMeta.Labels[pipelineloop.GroupName+pipelineLoopLabelKey] = pipelineLoopMeta.Name

	// Propagate annotations from PipelineLoop to Run.
	if customRun.ObjectMeta.Annotations == nil {
		customRun.ObjectMeta.Annotations = make(map[string]string, len(pipelineLoopMeta.Annotations))
	}
	for key, value := range pipelineLoopMeta.Annotations {
		customRun.ObjectMeta.Annotations[key] = value
	}
}

func storePipelineLoopSpec(status *pipelineloopv1alpha1.PipelineLoopRunStatus, tls *pipelineloopv1alpha1.PipelineLoopSpec) {
	// Only store the PipelineLoopSpec once, if it has never been set before.
	if status.PipelineLoopSpec == nil {
		status.PipelineLoopSpec = tls
	}
}

// Storing PipelineSpec and TaskSpec in PipelineRunStatus is a source of significant memory consumption and OOM failures.
// Additionally, performance of status update in customRun reconciler is impacted.
// PipelineSpec and TaskSpec seems to be redundant in this place.
// See issue: https://github.com/kubeflow/kfp-tekton/issues/962
func getPipelineRunStatusWithoutPipelineSpec(status *tektonv1.PipelineRunStatus) *tektonv1.PipelineRunStatus {
	s := status.DeepCopy()
	s.PipelineSpec = nil
	return s
}
