package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/peterhellberg/duration"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/termination"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	ArgoCompleteLabelKey   string = "workflows.argoproj.io/completed"
	MetadataExecutionIDKey string = "pipelines.kubeflow.org/metadata_execution_id"
	MaxCacheStalenessKey   string = "pipelines.kubeflow.org/max_cache_staleness"
)

func WatchPods(namespaceToWatch string, clientManager ClientManagerInterface) {
	zapLog, _ := zap.NewProduction()
	logger := zapLog.Sugar()
	defer zapLog.Sync()

	k8sCore := clientManager.KubernetesCoreClient()

	for {
		listOptions := metav1.ListOptions{
			Watch:         true,
			LabelSelector: CacheIDLabelKey,
		}
		watcher, err := k8sCore.PodClient(namespaceToWatch).Watch(context.Background(), listOptions)

		if err != nil {
			logger.Errorf("Watcher error: %v", err)
		}

		for event := range watcher.ResultChan() {
			pod := reflect.ValueOf(event.Object).Interface().(*corev1.Pod)
			if event.Type == watch.Error {
				logger.Errorf("Watcher error in loop: %v", event.Type)
				continue
			}
			log.Printf((*pod).GetName())

			if !isPodCompletedAndSucceeded(pod) {
				logger.Warnf("Pod %s is not completed or not in successful status, skip the loop.", pod.ObjectMeta.Name)
				continue
			}

			if isCacheWriten(pod.ObjectMeta.Labels) {
				logger.Warnf("Pod %s is already changed by cache, skip the loop.", pod.ObjectMeta.Name)
				continue
			}

			executionKey, exists := pod.ObjectMeta.Annotations[ExecutionKey]
			if !exists {
				logger.Errorf("Pod %s has no annotation: pipelines.kubeflow.org/execution_cache_key set, skip the loop.", pod.ObjectMeta.Name)

				continue
			}

			executionOutput, err := parseResult(pod, logger)
			if err != nil {
				logger.Errorf("Result of Pod %s not parse success.", pod.ObjectMeta.Name)
				continue
			}

			executionOutputMap := make(map[string]interface{})
			executionOutputMap[TektonTaskrunOutputs] = executionOutput
			executionOutputMap[MetadataExecutionIDKey] = pod.ObjectMeta.Labels[MetadataExecutionIDKey]
			executionOutputJSON, _ := json.Marshal(executionOutputMap)

			executionMaxCacheStaleness, exists := pod.ObjectMeta.Annotations[MaxCacheStalenessKey]
			var maxCacheStalenessInSeconds int64 = -1
			if exists {
				maxCacheStalenessInSeconds = getMaxCacheStaleness(executionMaxCacheStaleness)
			}

			executionTemplate := pod.ObjectMeta.Annotations[TektonTaskrunTemplate]
			executionToPersist := model.ExecutionCache{
				ExecutionCacheKey: executionKey,
				ExecutionTemplate: executionTemplate,
				ExecutionOutput:   string(executionOutputJSON),
				MaxCacheStaleness: maxCacheStalenessInSeconds,
			}

			cacheEntryCreated, err := clientManager.CacheStore().CreateExecutionCache(&executionToPersist)
			if err != nil {
				logger.Errorf("Unable to create cache entry for Pod: %s", pod.ObjectMeta.Name)
				continue
			}

			err = patchCacheID(k8sCore, pod, namespaceToWatch, cacheEntryCreated.ID, logger)
			if err != nil {
				logger.Errorf("Patch Pod: %s failed", pod.ObjectMeta.Name)
			}
		}
	}
}

func parseResult(pod *corev1.Pod, logger *zap.SugaredLogger) (string, error) {
	logger.Info("Start parse result from pod.")

	output := []*v1beta1.TaskRunResult{}

	containersState := pod.Status.ContainerStatuses
	if containersState == nil || len(containersState) == 0 {
		return "", fmt.Errorf("No container status found")
	}

	for _, state := range containersState {
		if state.State.Terminated != nil && len(state.State.Terminated.Message) != 0 {
			msg := state.State.Terminated.Message
			results, err := termination.ParseMessage(logger, msg)
			if err != nil {
				logger.Errorf("termination message could not be parsed as JSON: %v", err)
				return "", fmt.Errorf("termination message could not be parsed as JSON: %v", err)
			}

			for _, r := range results {
				if r.ResultType == v1beta1.TaskRunResultType {
					itemRes := v1beta1.TaskRunResult{}
					itemRes.Name = r.Key
					itemRes.Value = r.Value
					output = append(output, &itemRes)
				}
			}

			// assumption only on step in a task
			break
		}
	}

	if len(output) == 0 {
		logger.Errorf("No validate result found in pod.Status.ContainerStatuses[].State.Terminated.Message")
		return "", fmt.Errorf("No result found in the pod")
	}

	b, err := json.Marshal(output)
	if err != nil {
		logger.Errorf("Result marshl failed")
		return "", err
	}

	return string(b), nil
}

func isPodCompletedAndSucceeded(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded
}

func isCacheWriten(labels map[string]string) bool {
	cacheID := labels[CacheIDLabelKey]
	return cacheID != ""
}

func patchCacheID(k8sCore client.KubernetesCoreInterface, podToPatch *corev1.Pod, namespaceToWatch string, id int64, logger *zap.SugaredLogger) error {
	labels := podToPatch.ObjectMeta.Labels
	labels[CacheIDLabelKey] = strconv.FormatInt(id, 10)
	logger.Infof("Cache id: %d", id)

	var patchOps []patchOperation
	patchOps = append(patchOps, patchOperation{
		Op:    OperationTypeAdd,
		Path:  LabelPath,
		Value: labels,
	})
	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		logger.Errorf("Marshal patch for pod: %s failed", podToPatch.ObjectMeta.Name)
		return fmt.Errorf("Unable to patch cache_id to pod: %s", podToPatch.ObjectMeta.Name)
	}
	_, err = k8sCore.PodClient(namespaceToWatch).Patch(context.Background(), podToPatch.ObjectMeta.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		logger.Errorf("Unable to patch cache_id to pod: %s", podToPatch.ObjectMeta.Name)
		return err
	}

	logger.Info("Cache id patched.")
	return nil
}

// Convert RFC3339 Duration(Eg. "P1DT30H4S") to int64 seconds.
func getMaxCacheStaleness(maxCacheStaleness string) int64 {
	var seconds int64 = -1
	if d, err := duration.Parse(maxCacheStaleness); err == nil {
		seconds = int64(d / time.Second)
	}
	return seconds
}
