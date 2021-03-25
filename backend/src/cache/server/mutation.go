// Copyright 2020 Google LLC
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

package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KFPCacheEnabledLabelKey   string = "pipelines.kubeflow.org/cache_enabled"
	KFPCacheEnabledLabelValue string = "true"
	KFPCachedLabelKey         string = "pipelines.kubeflow.org/reused_from_cache"
	KFPCachedLabelValue       string = "true"
	ArgoWorkflowNodeName      string = "workflows.argoproj.io/node-name"
	TektonTaskrunTemplate     string = "tekton.dev/template"
	ExecutionKey              string = "pipelines.kubeflow.org/execution_cache_key"
	CacheIDLabelKey           string = "pipelines.kubeflow.org/cache_id"
	TektonTaskrunOutputs      string = "tekton.dev/outputs"
	MetadataWrittenKey        string = "pipelines.kubeflow.org/metadata_written"
	AnnotationPath            string = "/metadata/annotations"
	LabelPath                 string = "/metadata/labels"
	SpecContainersPath        string = "/spec/containers"
	SpecInitContainersPath    string = "/spec/initContainers"
	TFXPodSuffix              string = "tfx/orchestration/kubeflow/container_entrypoint.py"
	ArchiveLocationKey        string = "archiveLocation"
	TaskName                  string = "tekton.dev/pipelineTask"
	PipelineName              string = "pipelines.kubeflow.org/pipelinename"
	Generation                string = "pipelines.kubeflow.org/generation"

	TektonGroup        string = "tekton.dev/v1beta1"
	TektonTaskKind     string = "TaskRun"
	ToolInitContainner string = "place-tools"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

type ClientManagerInterface interface {
	CacheStore() storage.ExecutionCacheStoreInterface
	KubernetesCoreClient() client.KubernetesCoreInterface
	TektonClient() client.TektonInterface
}

type Template struct {
	Spec         tektonv1beta1.TaskRunSpec
	TaskName     string
	PipelineName string
	Generation   string
}

// MutatePodIfCached will check whether the execution has already been run before from MLMD
func MutatePodIfCached(req *v1beta1.AdmissionRequest, clientMgr ClientManagerInterface) ([]patchOperation, error) {
	zapLog, _ := zap.NewProduction()
	logger := zapLog.Sugar()
	defer zapLog.Sync()

	logger.Infof("Request received: %#v", req.Resource)
	// This handler should only get called on Pod objects as per the MutatingWebhookConfiguration in the YAML file.
	// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
	// let the object request pass through otherwise.
	if req.Resource != podResource {
		logger.Errorf("Expect resource to be %#v, but found %#v", podResource, req.Resource)
		return nil, nil
	}

	// Parse the Pod object.
	raw := req.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}

	// Pod filtering to only cache KFP tekton pods except TFX pods
	// TODO: Switch to objectSelector once Kubernetes 1.15 hits the GKE stable channel. See
	// https://github.com/kubernetes/kubernetes/pull/78505
	// https://cloud.google.com/kubernetes-engine/docs/release-notes-stable
	if !isKFPCacheEnabled(&pod, logger) {
		logger.Errorf("This pod %s does not enable cache.", pod.ObjectMeta.Name)
		return nil, nil
	}

	// Check if label[pipelines.kubeflow.org/pipelinename] and label[pipelines.kubeflow.org/generation]
	if !isLabeledCorrect(&pod, logger) {
		logger.Errorf("This pod %s is not labeled correct.", pod.ObjectMeta.Name)
		return nil, nil
	}

	// TODO, TFXPod? do some research later
	if isTFXPod(&pod, logger) {
		logger.Errorf("This pod %s is created by tfx pipelines.", pod.ObjectMeta.Name)
		return nil, nil
	}

	// Check if it's a taskrun owned pod
	trName, isTaskrunOwned := isTaskrunOwn(&pod)
	if !isTaskrunOwned {
		logger.Errorf("This pod %s is not owned by Taskrun.", pod.ObjectMeta.Name)
		return nil, nil
	}

	if isV2Pod(&pod) {
		// KFP v2 handles caching by its driver.
		log.Printf("This pod %s is created by KFP v2 pipelines.", pod.ObjectMeta.Name)
		return nil, nil
	}

	var patches []patchOperation
	annotations := pod.ObjectMeta.Annotations
	labels := pod.ObjectMeta.Labels
	_, exists := annotations[TektonTaskrunTemplate]
	var executionHashKey string
	if !exists {
		logger.Errorf("This pod %s has no annotation: tekton.dev/template.", pod.ObjectMeta.Name)
		return patches, nil
	}

	// get owner taskrun for the pod, for spec and taskname to calculate the hashcode.
	tr, err := clientMgr.TektonClient().GetTaskRun(pod.Namespace, trName, metav1.GetOptions{})
	if err != nil {
		logger.Errorf("Unable to get Taskrun which own the Pod %s/%s : %v", pod.ObjectMeta.Namespace, trName, err)
		return patches, nil
	}

	// Generate the executionHashKey based on Taskrun.status.taskspec and the name of task
	executionHashKey, template, err := generateCacheKeyFromTemplate(tr, &pod)
	if err != nil {
		logger.Errorf("Unable to generate cache key for pod %s : %v", pod.ObjectMeta.Name, err)
		return patches, nil
	}

	annotations[TektonTaskrunTemplate] = template
	annotations[ExecutionKey] = executionHashKey
	labels[CacheIDLabelKey] = ""
	var maxCacheStalenessInSeconds int64 = -1
	maxCacheStaleness, exists := annotations[MaxCacheStalenessKey]
	if exists {
		maxCacheStalenessInSeconds = getMaxCacheStaleness(maxCacheStaleness)
	}

	var cachedExecution *model.ExecutionCache
	cachedExecution, err = clientMgr.CacheStore().GetExecutionCache(executionHashKey, maxCacheStalenessInSeconds)
	if err != nil {
		logger.Warnf("Failed when try to get cache from storage: %v", err)
	}
	// Found cached execution, add cached output and cache_id and replace container images.
	if cachedExecution != nil {
		logger.Infof("Cached output: " + cachedExecution.ExecutionOutput)

		result := getValueFromSerializedMap(cachedExecution.ExecutionOutput, TektonTaskrunOutputs)
		annotations[TektonTaskrunOutputs] = result
		labels[CacheIDLabelKey] = strconv.FormatInt(cachedExecution.ID, 10)
		labels[KFPCachedLabelKey] = KFPCachedLabelValue // This label indicates the pod is taken from cache.

		// These labels cache results for metadata-writer.
		labels[MetadataExecutionIDKey] = getValueFromSerializedMap(cachedExecution.ExecutionOutput, MetadataExecutionIDKey)
		labels[MetadataWrittenKey] = "true"

		dummyContainers, err := prepareMainContainer(&pod, result, logger)
		if err != nil || len(dummyContainers) == 0 {
			logger.Errorf("Unable prepare dummy container %s : %v", pod.ObjectMeta.Name, err)
			return patches, nil
		}

		patches = append(patches, patchOperation{
			Op:    OperationTypeReplace,
			Path:  SpecContainersPath,
			Value: dummyContainers,
		})

		// Handler init containers, do or not?
		if pod.Spec.InitContainers != nil || len(pod.Spec.InitContainers) != 0 {
			dummyInitContainers, err := prepareInitContainer(&pod, logger)
			if err != nil {
				logger.Errorf("Unable prepare dummy init container %s : %v", pod.ObjectMeta.Name, err)
				return patches, nil
			}

			patches = append(patches, patchOperation{
				Op:    OperationTypeReplace,
				Path:  SpecInitContainersPath,
				Value: dummyInitContainers,
			})
		}
	}

	// Add executionKey to pod.metadata.annotations
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  AnnotationPath,
		Value: annotations,
	})

	// Add cache_id label key
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  LabelPath,
		Value: labels,
	})

	return patches, nil
}

func prepareInitContainer(pod *corev1.Pod, logger *zap.SugaredLogger) ([]corev1.Container, error) {
	logger.Infof("Start to prepare dummy init containers.")
	dummyContainers := []corev1.Container{}

	for _, container := range pod.Spec.InitContainers {
		if container.Name == ToolInitContainner {
			dummyContainers = append(dummyContainers, container)
		}
	}

	return dummyContainers, nil
}

func prepareMainContainer(pod *corev1.Pod, result string, logger *zap.SugaredLogger) ([]corev1.Container, error) {
	logger.Infof("Start to prepare dummy containers.")
	dummyContainers := []corev1.Container{}

	results, err := unmarshalResult(result)
	if err != nil {
		logger.Errorf("Unmarshal result of taskrun failed: ", err)
		return dummyContainers, err
	}

	args := []string{}
	for _, result := range results {
		arg := fmt.Sprintf("printf '%s' | tee /tekton/results/%s", result.Value, result.Name)
		args = append(args, arg)
	}

	replacedArg := strings.Join(args, ";")

	argStartFlag := -1
	// assumptive there is only one container in section container
	firstOriginalContainer := pod.Spec.Containers[0]

	for index, arg := range firstOriginalContainer.Args {
		if arg == "--" {
			argStartFlag = index
		}
	}

	firstOriginalContainer.Args = append(firstOriginalContainer.Args[:argStartFlag-1], "/bin/bash")
	firstOriginalContainer.Args = append(firstOriginalContainer.Args, "--")
	firstOriginalContainer.Args = append(firstOriginalContainer.Args, "-c")
	firstOriginalContainer.Args = append(firstOriginalContainer.Args, replacedArg)
	firstOriginalContainer.Image = "registry.access.redhat.com/ubi8/ubi-minimal"

	dummyContainers = append(dummyContainers, firstOriginalContainer)

	return dummyContainers, nil
}

func unmarshalResult(taskResult string) ([]tektonv1beta1.TaskRunResult, error) {
	var results []tektonv1beta1.TaskRunResult
	err := json.Unmarshal([]byte(taskResult), &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func generateCacheKeyFromTemplate(taskRun *tektonv1beta1.TaskRun, pod *corev1.Pod) (string, string, error) {
	template := Template{}
	template.Spec = taskRun.Spec
	template.TaskName = pod.ObjectMeta.Labels[TaskName]
	template.PipelineName = pod.ObjectMeta.Labels[PipelineName]
	template.Generation = pod.ObjectMeta.Labels[Generation]

	b, err := json.Marshal(template)
	if err != nil {
		return "", "", err
	}
	hash := sha256.New()
	hash.Write(b)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)

	return executionHashKey, string(b), nil
}

func getValueFromSerializedMap(serializedMap string, key string) string {
	var outputMap map[string]interface{}
	b := []byte(serializedMap)
	err := json.Unmarshal(b, &outputMap)
	if err != nil {
		return ""
	}

	value, exist := outputMap[key].(string)
	if !exist || value == "" {
		return ""
	}

	return value
}

func isKFPCacheEnabled(pod *corev1.Pod, logger *zap.SugaredLogger) bool {
	cacheEnabled, exists := pod.ObjectMeta.Labels[KFPCacheEnabledLabelKey]
	if !exists {
		logger.Errorf("This pod %s is not created by KFP.", pod.ObjectMeta.Name)
		return false
	}
	return cacheEnabled == KFPCacheEnabledLabelValue
}

func isLabeledCorrect(pod *corev1.Pod, logger *zap.SugaredLogger) bool {
	_, exists := pod.ObjectMeta.Labels[PipelineName]
	if !exists {
		logger.Errorf("This pod %s is not label the name of pipeline.", pod.ObjectMeta.Name)
		return false
	}
	_, exists = pod.ObjectMeta.Labels[Generation]
	if !exists {
		logger.Errorf("This pod %s is not label the generation of pipeline.", pod.ObjectMeta.Name)
		return false
	}

	return true
}

func isTFXPod(pod *corev1.Pod, logger *zap.SugaredLogger) bool {
	containers := pod.Spec.Containers
	if containers == nil || len(containers) == 0 {
		logger.Info("This pod container does not exist.")
		return true
	}
	var mainContainers []corev1.Container
	for _, c := range containers {
		if c.Name != "" && c.Name == "main" {
			mainContainers = append(mainContainers, c)
		}
	}
	if len(mainContainers) != 1 {
		return false
	}
	mainContainer := mainContainers[0]
	return len(mainContainer.Command) != 0 && strings.HasSuffix(mainContainer.Command[len(mainContainer.Command)-1], TFXPodSuffix)
}

func isTaskrunOwn(pod *corev1.Pod) (string, bool) {
	for _, ref := range pod.GetOwnerReferences() {
		if ref.Kind == TektonTaskKind && ref.APIVersion == TektonGroup {
			return ref.Name, true
		}
	}

	return "", false
}
