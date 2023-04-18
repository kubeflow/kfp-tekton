// Copyright 2020 kubeflow.org
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

package util

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	exec "github.com/kubeflow/pipelines/backend/src/common"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	pipelineapiv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/run/v1alpha1"
	customRun "github.com/tektoncd/pipeline/pkg/apis/run/v1beta1"
	prclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	prclientv1beta1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	prsinformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	prinformer "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/cache"
)

// PipelineRun is a type to help manipulate PipelineRun objects.
type PipelineRun struct {
	*pipelineapi.PipelineRun
}

type runKinds []string

var (
	// A list of Kinds that contains childReferences
	// those childReferences would be scaned and retrieve their taskrun/run status
	childReferencesKinds runKinds = []string{}
)

const (
	childReferencesKindFlagName = "childReferencesKinds"
)

func (rk *runKinds) String() string {
	return fmt.Sprint(*rk)
}

func (rk *runKinds) Set(value string) error {
	if len(*rk) > 0 {
		return fmt.Errorf("%s has been set", childReferencesKindFlagName)
	}

	for _, k := range strings.Split(value, ",") {
		*rk = append(*rk, k)
	}
	sort.Strings(*rk)

	return nil
}

func init() {
	flag.Var(&childReferencesKinds, childReferencesKindFlagName, "A list of kinds to search for the nested childReferences")
}

func NewPipelineRunFromBytes(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := yaml.Unmarshal(bytes, &pr)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	return NewPipelineRun(&pr), nil
}

func NewPipelineRunFromBytesJSON(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := json.Unmarshal(bytes, &pr)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	return NewPipelineRun(&pr), nil
}

func NewPipelineRunFromInterface(obj interface{}) (*PipelineRun, error) {
	pr, ok := obj.(*pipelineapi.PipelineRun)
	if ok {
		return NewPipelineRun(pr), nil
	}
	return nil, NewInvalidInputError("not PipelineRun struct")
}

func UnmarshParametersPipelineRun(paramsString string) (SpecParameters, error) {
	if paramsString == "" {
		return nil, nil
	}
	var params []pipelineapi.Param
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, NewInternalServerError(err, "Parameters have wrong format")
	}
	rev := make(SpecParameters, 0, len(params))
	for _, param := range params {
		rev = append(rev, SpecParameter{
			Name:  param.Name,
			Value: StringPointer(param.Value.StringVal)})
	}
	return rev, nil
}

func MarshalParametersPipelineRun(params SpecParameters) (string, error) {
	if params == nil {
		return "[]", nil
	}

	inputParams := make([]pipelineapi.Param, 0)
	for _, param := range params {
		newParam := pipelineapi.Param{
			Name:  param.Name,
			Value: pipelineapi.ArrayOrString{Type: "string", StringVal: *param.Value},
		}
		inputParams = append(inputParams, newParam)
	}
	paramBytes, err := json.Marshal(inputParams)
	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

func NewPipelineRunFromScheduleWorkflowSpecBytesJSON(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := json.Unmarshal(bytes, &pr.Spec)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	pr.APIVersion = "tekton.dev/v1beta1"
	pr.Kind = "PipelineRun"
	return NewPipelineRun(&pr), nil
}

// NewWorkflow creates a Workflow.
func NewPipelineRun(pr *pipelineapi.PipelineRun) *PipelineRun {
	return &PipelineRun{
		pr,
	}
}

func (w *PipelineRun) GetWorkflowParametersAsMap() map[string]string {
	resultAsArray := w.Spec.Params
	resultAsMap := make(map[string]string)
	for _, param := range resultAsArray {
		resultAsMap[param.Name] = param.Value.StringVal
	}
	return resultAsMap
}

// SetServiceAccount Set the service account to run the workflow.
func (pr *PipelineRun) SetServiceAccount(serviceAccount string) {
	pr.Spec.ServiceAccountName = serviceAccount
}

// OverrideParameters overrides some of the parameters of a Workflow.
func (pr *PipelineRun) OverrideParameters(desiredParams map[string]string) {
	desiredSlice := make([]pipelineapi.Param, 0)
	for _, currentParam := range pr.Spec.Params {
		var desiredValue pipelineapi.ArrayOrString = pipelineapi.ArrayOrString{
			Type:      "string",
			StringVal: "",
		}
		if param, ok := desiredParams[currentParam.Name]; ok {
			desiredValue.StringVal = param
		} else {
			desiredValue.StringVal = currentParam.Value.StringVal
		}
		desiredSlice = append(desiredSlice, pipelineapi.Param{
			Name:  currentParam.Name,
			Value: desiredValue,
		})
	}
	pr.Spec.Params = desiredSlice
}

func (pr *PipelineRun) VerifyParameters(desiredParams map[string]string) error {
	templateParamsMap := make(map[string]*string)
	for _, param := range pr.Spec.Params {
		templateParamsMap[param.Name] = &param.Value.StringVal
	}
	for k := range desiredParams {
		_, ok := templateParamsMap[k]
		if !ok {
			glog.Warningf("Unrecognized input parameter: %v", k)
		}
	}
	return nil
}

// Get converts this object to a workflowapi.Workflow.
func (pr *PipelineRun) Get() *pipelineapi.PipelineRun {
	return pr.PipelineRun
}

func (pr *PipelineRun) ScheduledWorkflowUUIDAsStringOrEmpty() string {
	if pr.OwnerReferences == nil {
		return ""
	}

	for _, reference := range pr.OwnerReferences {
		if isScheduledWorkflow(reference) {
			return string(reference.UID)
		}
	}

	return ""
}

func (pr *PipelineRun) ScheduledAtInSecOr0() int64 {
	if pr.Labels == nil {
		return 0
	}

	for key, value := range pr.Labels {
		if key == LabelKeyWorkflowEpoch {
			result, err := RetrieveInt64FromLabel(value)
			if err != nil {
				glog.Errorf("Could not retrieve scheduled epoch from label key (%v) and label value (%v).", key, value)
				return 0
			}
			return result
		}
	}

	return 0
}

func (pr *PipelineRun) FinishedAt() int64 {
	if pr.Status.PipelineRunStatusFields.CompletionTime.IsZero() {
		// If workflow is not finished
		return 0
	}
	return pr.Status.PipelineRunStatusFields.CompletionTime.Unix()
}

func (pr *PipelineRun) FinishedAtTime() metav1.Time {
	return *pr.Status.PipelineRunStatusFields.CompletionTime
}

func (pr *PipelineRun) Condition() exec.ExecutionPhase {
	if len(pr.Status.Status.Conditions) > 0 {
		switch pr.Status.Status.Conditions[0].Reason {
		case "Error":
			return exec.ExecutionError
		case "Failed":
			return exec.ExecutionFailed
		case "InvalidTaskResultReference":
			return exec.ExecutionFailed
		case "Cancelled":
			return exec.ExecutionFailed
		case "Pending":
			return exec.ExecutionPending
		case "Running":
			return exec.ExecutionRunning
		case "Succeeded":
			return exec.ExecutionSucceeded
		case "Completed":
			return exec.ExecutionSucceeded
		case "PipelineRunTimeout":
			return exec.ExecutionError
		case "PipelineRunCancelled":
			return exec.ExecutionPhase("Terminated")
		case "PipelineRunCouldntCancel":
			return exec.ExecutionError
		case "Terminating":
			return exec.ExecutionPhase("Terminating")
		case "Terminated":
			return exec.ExecutionPhase("Terminated")
		default:
			return exec.ExecutionUnknown
		}
	} else {
		return ""
	}
}

func (pr *PipelineRun) ToStringForStore() string {
	workflow, err := json.Marshal(pr.PipelineRun)
	if err != nil {
		glog.Errorf("Could not marshal the workflow: %v", pr.PipelineRun)
		return ""
	}
	return string(workflow)
}

func (pr *PipelineRun) HasScheduledWorkflowAsParent() bool {
	return containsScheduledWorkflow(pr.PipelineRun.OwnerReferences)
}

func (pr *PipelineRun) GetExecutionSpec() ExecutionSpec {
	pipelinerun := pr.DeepCopy()
	pipelinerun.Status = pipelineapi.PipelineRunStatus{}
	pipelinerun.TypeMeta = metav1.TypeMeta{Kind: pr.Kind, APIVersion: pr.APIVersion}
	// To prevent collisions, clear name, set GenerateName to first 200 runes of previous name.
	nameRunes := []rune(pr.Name)
	length := len(nameRunes)
	if length > 200 {
		length = 200
	}
	pipelinerun.ObjectMeta = metav1.ObjectMeta{GenerateName: string(nameRunes[:length])}
	return NewPipelineRun(pipelinerun)
}

// OverrideName sets the name of a Workflow.
func (pr *PipelineRun) OverrideName(name string) {
	pr.GenerateName = ""
	pr.Name = name
}

// SetAnnotationsToAllTemplatesIfKeyNotExist sets annotations on all templates in a Workflow
// if the annotation key does not exist
func (pr *PipelineRun) SetAnnotationsToAllTemplatesIfKeyNotExist(key string, value string) {
	// No metadata object within pipelineRun task
}

// SetLabels sets labels on all templates in a Workflow
func (pr *PipelineRun) SetLabelsToAllTemplates(key string, value string) {
	// No metadata object within pipelineRun task
}

// SetOwnerReferences sets owner references on a Workflow.
func (pr *PipelineRun) SetOwnerReferences(schedule *swfapi.ScheduledWorkflow) {
	pr.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(schedule, schema.GroupVersionKind{
			Group:   swfapi.SchemeGroupVersion.Group,
			Version: swfapi.SchemeGroupVersion.Version,
			Kind:    swfregister.Kind,
		}),
	}
}

func (pr *PipelineRun) SetLabels(key string, value string) {
	if pr.Labels == nil {
		pr.Labels = make(map[string]string)
	}
	pr.Labels[key] = value
}

func (pr *PipelineRun) SetAnnotations(key string, value string) {
	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations[key] = value
}

func (pr *PipelineRun) ReplaceUID(id string) error {
	newWorkflowString := strings.Replace(pr.ToStringForStore(), "{{workflow.uid}}", id, -1)
	newWorkflowString = strings.Replace(newWorkflowString, "$(context.pipelineRun.uid)", id, -1)
	var workflow *pipelineapi.PipelineRun
	if err := json.Unmarshal([]byte(newWorkflowString), &workflow); err != nil {
		return NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow: %s", pr.ToStringForStore())
	}
	pr.PipelineRun = workflow
	return nil
}

func (pr *PipelineRun) ReplaceOrignalPipelineRunName(name string) error {
	newWorkflowString := strings.Replace(pr.ToStringForStore(), "$ORIG_PR_NAME", name, -1)
	var workflow *pipelineapi.PipelineRun
	if err := json.Unmarshal([]byte(newWorkflowString), &workflow); err != nil {
		return NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow: %s", pr.ToStringForStore())
	}
	pr.PipelineRun = workflow
	return nil
}

func (pr *PipelineRun) SetCannonicalLabels(name string, nextScheduledEpoch int64, index int64) {
	pr.SetLabels(LabelKeyWorkflowScheduledWorkflowName, name)
	pr.SetLabels(LabelKeyWorkflowEpoch, FormatInt64ForLabel(nextScheduledEpoch))
	pr.SetLabels(LabelKeyWorkflowIndex, FormatInt64ForLabel(index))
	pr.SetLabels(LabelKeyWorkflowIsOwnedByScheduledWorkflow, "true")
}

// FindObjectStoreArtifactKeyOrEmpty loops through all node running statuses and look up the first
// S3 artifact with the specified nodeID and artifactName. Returns empty if nothing is found.
func (pr *PipelineRun) FindObjectStoreArtifactKeyOrEmpty(nodeID string, artifactName string) string {
	// TODO: The below artifact keys are only for parameter artifacts. Will need to also implement
	//       metric and raw input artifacts once we finallized the big data passing in our compiler.

	if pr.Status.PipelineRunStatusFields.TaskRuns == nil {
		return ""
	}
	return "artifacts/" + pr.ObjectMeta.Name + "/" + nodeID + "/" + artifactName + ".tgz"
}

// FindTaskRunByPodName loops through all workflow task runs and look up by the pod name.
func (pr *PipelineRun) FindTaskRunByPodName(podName string) (*pipelineapi.PipelineRunTaskRunStatus, string) {
	for id, taskRun := range pr.Status.TaskRuns {
		if taskRun.Status.PodName == podName {
			return taskRun, id
		}
	}
	return nil, ""
}

// IsInFinalState whether the workflow is in a final state.
func (pr *PipelineRun) IsInFinalState() bool {
	// Workflows in the statuses other than pending or running are considered final.

	if len(pr.Status.Status.Conditions) > 0 {
		finalConditions := map[string]int{
			"Succeeded":                  1,
			"Failed":                     1,
			"Completed":                  1,
			"PipelineRunCancelled":       1, // remove this when Tekton move to v1 API
			"PipelineRunCouldntCancel":   1,
			"PipelineRunTimeout":         1,
			"Cancelled":                  1,
			"StoppedRunFinally":          1,
			"CancelledRunFinally":        1,
			"InvalidTaskResultReference": 1,
		}
		phase := pr.Status.Status.Conditions[0].Reason
		if _, ok := finalConditions[phase]; ok {
			return true
		}
	}
	return false
}

// PersistedFinalState whether the workflow final state has being persisted.
func (pr *PipelineRun) PersistedFinalState() bool {
	if _, ok := pr.GetLabels()[LabelKeyWorkflowPersistedFinalState]; ok {
		// If the label exist, workflow final state has being persisted.
		return true
	}
	return false
}

// IsV2Compatible whether the workflow is a v2 compatible pipeline.
func (pr *PipelineRun) IsV2Compatible() bool {
	value := pr.GetObjectMeta().GetAnnotations()["pipelines.kubeflow.org/v2_pipeline"]
	return value == "true"
}

// no compression/decompression in tekton
func (pr *PipelineRun) Decompress() error {
	return nil
}

// Always can retry
func (pr *PipelineRun) CanRetry() error {
	return nil
}

func (pr *PipelineRun) ExecutionName() string {
	return pr.Name
}

func (pr *PipelineRun) SetExecutionName(name string) {
	pr.GenerateName = ""
	pr.Name = name

}

func (pr *PipelineRun) ExecutionNamespace() string {
	return pr.Namespace
}

func (pr *PipelineRun) SetExecutionNamespace(namespace string) {
	pr.Namespace = namespace
}

func (pr *PipelineRun) ExecutionObjectMeta() *metav1.ObjectMeta {
	return &pr.ObjectMeta
}

func (pr *PipelineRun) ExecutionTypeMeta() *metav1.TypeMeta {
	return &pr.TypeMeta
}

func (pr *PipelineRun) ExecutionStatus() ExecutionStatus {
	return pr
}

func (pr *PipelineRun) ExecutionType() ExecutionType {
	return TektonPipelineRun
}

func (pr *PipelineRun) ExecutionUID() string {
	return string(pr.UID)
}

func (pr *PipelineRun) HasMetrics() bool {
	return pr.Status.PipelineRunStatusFields.TaskRuns != nil
}

func (pr *PipelineRun) Message() string {
	if pr.Status.Conditions != nil && len(pr.Status.Conditions) > 0 {
		return pr.Status.Conditions[0].Message
	}
	return ""
}

func (pr *PipelineRun) StartedAtTime() metav1.Time {
	return *pr.Status.PipelineRunStatusFields.StartTime
}

func (pr *PipelineRun) IsTerminating() bool {
	return pr.Spec.Status == "Cancelled" && !pr.IsDone()
}

func (pr *PipelineRun) ServiceAccount() string {
	return pr.Spec.ServiceAccountName
}

func (pr *PipelineRun) SetPodMetadataLabels(key string, value string) {
	if pr.Labels == nil {
		pr.Labels = make(map[string]string)
	}
	pr.Labels[key] = value
}

func (pr *PipelineRun) SetSpecParameters(params SpecParameters) {
	desiredSlice := make([]pipelineapi.Param, 0)
	for _, currentParam := range params {
		newParam := pipelineapi.Param{
			Name: currentParam.Name,
			Value: pipelineapi.ArrayOrString{
				Type:      "string",
				StringVal: *currentParam.Value,
			},
		}
		desiredSlice = append(desiredSlice, newParam)
	}
	pr.Spec.Params = desiredSlice
}

func (pr *PipelineRun) Version() string {
	return pr.ResourceVersion
}

func (pr *PipelineRun) SetVersion(version string) {
	pr.ResourceVersion = version
}

func (pr *PipelineRun) SpecParameters() SpecParameters {
	rev := make(SpecParameters, 0, len(pr.Spec.Params))
	for _, currentParam := range pr.Spec.Params {
		rev = append(rev, SpecParameter{
			Name:  currentParam.Name,
			Value: StringPointer(currentParam.Value.StringVal)})
	}
	return rev
}

func (pr *PipelineRun) ToStringForSchedule() string {
	spec, err := json.Marshal(pr.PipelineRun.Spec)
	if err != nil {
		glog.Errorf("Could not marshal the Spec of workflow: %v", pr.PipelineRun)
		return ""
	}
	return string(spec)
}

func (w *PipelineRun) Validate(lint, ignoreEntrypoint bool) error {
	return nil
}

func (pr *PipelineRun) GenerateRetryExecution() (ExecutionSpec, []string, error) {
	if len(pr.Status.Status.Conditions) > 0 {
		switch pr.Status.Status.Conditions[0].Type {
		case "Failed", "Error":
			break
		default:
			return nil, nil, NewBadRequestError(errors.New("workflow cannot be retried"), "Workflow must be Failed/Error to retry")
		}
	}

	// TODO: Fix the below code to retry Tekton task. It may not be possible with the
	//       current implementation because Tekton doesn't have the concept of pipeline
	//       phases.

	newWF := pr.DeepCopy()
	// Delete/reset fields which indicate workflow completed
	// delete(newWF.Labels, common.LabelKeyCompleted)
	// // Delete/reset fields which indicate workflow is finished being persisted to the database
	// delete(newWF.Labels, util.LabelKeyWorkflowPersistedFinalState)
	// newWF.ObjectMeta.Labels[common.LabelKeyPhase] = string("Running")
	// newWF.Status.Phase = "Running"
	// newWF.Status.Message = ""
	// newWF.Status.FinishedAt = metav1.Time{}
	// if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
	// 	// if it was terminated, unset the deadline
	// 	newWF.Spec.ActiveDeadlineSeconds = nil
	// }

	// // Iterate the previous nodes. If it was successful Pod carry it forward
	// newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	// onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	var podsToDelete []string
	// for _, node := range wf.Status.Nodes {
	// 	switch node.Phase {
	// 	case "Succeeded", "Skipped":
	// 		if !strings.HasPrefix(node.Name, onExitNodeName) {
	// 			newWF.Status.Nodes[node.ID] = node
	// 			continue
	// 		}
	// 	case "Error", "Failed":
	// 		if !strings.HasPrefix(node.Name, onExitNodeName) && node.Type == "DAG" {
	// 			newNode := node.DeepCopy()
	// 			newNode.Phase = "Running"
	// 			newNode.Message = ""
	// 			newNode.FinishedAt = metav1.Time{}
	// 			newWF.Status.Nodes[newNode.ID] = *newNode
	// 			continue
	// 		}
	// 		// do not add this status to the node. pretend as if this node never existed.
	// 	default:
	// 		// Do not allow retry of workflows with pods in Running/Pending phase
	// 		return nil, nil, util.NewInternalServerError(
	// 			errors.New("workflow cannot be retried"),
	// 			"Workflow cannot be retried with node %s in %s phase", node.ID, node.Phase)
	// 	}
	// 	if node.Type == "Pod" {
	// 		podsToDelete = append(podsToDelete, node.ID)
	// 	}
	// }
	return NewPipelineRun(newWF), podsToDelete, nil
}

func (pr *PipelineRun) CollectionMetrics(retrieveArtifact RetrieveArtifact, user string) ([]*api.RunMetric, []error) {
	runID := pr.ObjectMeta.Labels[LabelKeyWorkflowRunId]
	runMetrics := []*api.RunMetric{}
	partialFailures := []error{}
	for _, taskrunStatus := range pr.Status.PipelineRunStatusFields.TaskRuns {
		nodeMetrics, err := collectTaskRunMetricsOrNil(runID, *taskrunStatus, retrieveArtifact, user)
		if err != nil {
			partialFailures = append(partialFailures, err)
			continue
		}
		if nodeMetrics != nil {
			if len(runMetrics)+len(nodeMetrics) >= maxMetricsCountLimit {
				leftQuota := maxMetricsCountLimit - len(runMetrics)
				runMetrics = append(runMetrics, nodeMetrics[0:leftQuota]...)
				// TODO(#1426): report the error back to api server to notify user
				log.Errorf("Reported metrics are more than the limit %v", maxMetricsCountLimit)
				break
			}
			runMetrics = append(runMetrics, nodeMetrics...)
		}
	}
	return runMetrics, partialFailures
}

func collectTaskRunMetricsOrNil(
	runID string, taskrunStatus pipelineapi.PipelineRunTaskRunStatus, retrieveArtifact RetrieveArtifact, user string) (
	[]*api.RunMetric, error) {

	defer func() {
		if panicMessage := recover(); panicMessage != nil {
			log.Infof("nodeStatus is not yet created. Panic message: '%v'.", panicMessage)
		}
	}()
	if taskrunStatus.Status == nil ||
		taskrunStatus.Status.TaskRunStatusFields.CompletionTime == nil {
		return nil, nil
	}
	metricsJSON, err := readTaskRunMetricsJSONOrEmpty(runID, taskrunStatus, retrieveArtifact, user)
	if err != nil || metricsJSON == "" {
		return nil, err
	}

	// Proto json lib requires a proto message before unmarshal data from JSON. We use
	// ReportRunMetricsRequest as a workaround to hold user's metrics, which is a superset of what
	// user can provide.
	reportMetricsRequest := new(api.ReportRunMetricsRequest)
	err = jsonpb.UnmarshalString(metricsJSON, reportMetricsRequest)
	if err != nil {
		// User writes invalid metrics JSON.
		// TODO(#1426): report the error back to api server to notify user
		log.WithFields(log.Fields{
			"run":         runID,
			"node":        taskrunStatus.PipelineTaskName,
			"raw_content": metricsJSON,
			"error":       err.Error(),
		}).Warning("Failed to unmarshal metrics file.")
		return nil, NewCustomError(err, CUSTOM_CODE_PERMANENT,
			"failed to unmarshal metrics file from (%s, %s).", runID, taskrunStatus.PipelineTaskName)
	}
	if reportMetricsRequest.GetMetrics() == nil {
		return nil, nil
	}
	for _, metric := range reportMetricsRequest.GetMetrics() {
		// User metrics just have name and value but no NodeId.
		metric.NodeId = taskrunStatus.PipelineTaskName
	}
	return reportMetricsRequest.GetMetrics(), nil
}

func readTaskRunMetricsJSONOrEmpty(
	runID string, nodeStatus pipelineapi.PipelineRunTaskRunStatus,
	retrieveArtifact RetrieveArtifact, user string) (string, error) {
	// Tekton doesn't support any artifact spec, artifact records are done by our custom metadata writers:
	// 	if nodeStatus.Outputs == nil || nodeStatus.Outputs.Artifacts == nil {
	// 		return "", nil // No output artifacts, skip the reporting
	// 	}
	// 	var foundMetricsArtifact bool = false
	// 	for _, artifact := range nodeStatus.Outputs.Artifacts {
	// 		if artifact.Name == metricsArtifactName {
	// 			foundMetricsArtifact = true
	// 		}
	// 	}
	// 	if !foundMetricsArtifact {
	// 		return "", nil // No metrics artifact, skip the reporting
	// 	}

	artifactRequest := &api.ReadArtifactRequest{
		RunId:        runID,
		NodeId:       nodeStatus.PipelineTaskName,
		ArtifactName: metricsArtifactName,
	}
	artifactResponse, err := retrieveArtifact(artifactRequest, user)
	if err != nil {
		return "", err
	}
	if artifactResponse == nil || artifactResponse.GetData() == nil || len(artifactResponse.GetData()) == 0 {
		// If artifact is not found or empty content, skip the reporting.
		return "", nil
	}
	archivedFiles, err := ExtractTgz(string(artifactResponse.GetData()))
	if err != nil {
		// Invalid tgz file. This should never happen unless there is a bug in the system and
		// it is a unrecoverable error.
		return "", NewCustomError(err, CUSTOM_CODE_PERMANENT,
			"Unable to extract metrics tgz file read from (%+v): %v", artifactRequest, err)
	}
	//There needs to be exactly one metrics file in the artifact archive. We load that file.
	if len(archivedFiles) == 1 {
		for _, value := range archivedFiles {
			return value, nil
		}
	}
	return "", NewCustomErrorf(CUSTOM_CODE_PERMANENT,
		"There needs to be exactly one metrics file in the artifact archive, but zero or multiple files were found.")
}

func (pr *PipelineRun) NodeStatuses() map[string]NodeStatus {
	// TODO: add implementation
	rev := make(map[string]NodeStatus)
	// rev := make(map[string]NodeStatus, len(pr.Status.PipelineRunStatusFields.ChildReferences))

	return rev
}

func (pr *PipelineRun) HasNodes() bool {
	return false
	// TODO: add implementation
	// return len(pr.Status.PipelineRunStatusFields.ChildReferences) > 0
}

// implementation of ExecutionClientInterface
type PipelineRunClient struct {
	client *prclientset.Clientset
}

func (prc *PipelineRunClient) Execution(namespace string) ExecutionInterface {
	var informer prinformer.PipelineRunInformer
	if namespace == "" {
		informer = prsinformers.NewSharedInformerFactory(prc.client, time.Second*30).
			Tekton().V1beta1().PipelineRuns()
	} else {
		informer = prsinformers.NewFilteredSharedInformerFactory(prc.client, time.Second*30, namespace, nil).
			Tekton().V1beta1().PipelineRuns()
	}

	return &PipelineRunInterface{
		pipelinerunInterface: prc.client.TektonV1beta1().PipelineRuns(namespace),
		informer:             informer,
	}
}

func (prc *PipelineRunClient) Compare(old, new interface{}) bool {
	newWorkflow := new.(*pipelineapi.PipelineRun)
	oldWorkflow := old.(*pipelineapi.PipelineRun)
	// Periodic resync will send update events for all known Workflows.
	// Two different versions of the same WorkflowHistory will always have different RVs.
	return newWorkflow.ResourceVersion != oldWorkflow.ResourceVersion
}

type PipelineRunInterface struct {
	pipelinerunInterface prclientv1beta1.PipelineRunInterface
	informer             prinformer.PipelineRunInformer
}

func (pri *PipelineRunInterface) Create(ctx context.Context, execution ExecutionSpec, opts metav1.CreateOptions) (ExecutionSpec, error) {
	pipelinerun, ok := execution.(*PipelineRun)
	if !ok {
		return nil, fmt.Errorf("execution is not a valid ExecutionSpec for Argo Workflow")
	}

	revPipelineRun, err := pri.pipelinerunInterface.Create(ctx, pipelinerun.PipelineRun, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun}, nil
}

func (pri *PipelineRunInterface) Update(ctx context.Context, execution ExecutionSpec, opts metav1.UpdateOptions) (ExecutionSpec, error) {
	pipelinerun, ok := execution.(*PipelineRun)
	if !ok {
		return nil, fmt.Errorf("execution is not a valid ExecutionSpec for Argo Workflow")
	}

	revPipelineRun, err := pri.pipelinerunInterface.Update(ctx, pipelinerun.PipelineRun, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun}, nil
}

func (pri *PipelineRunInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return pri.pipelinerunInterface.Delete(ctx, name, opts)
}

func (pri *PipelineRunInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return pri.pipelinerunInterface.DeleteCollection(ctx, opts, listOpts)
}

func (pri *PipelineRunInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (ExecutionSpec, error) {
	revPipelineRun, err := pri.pipelinerunInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun}, nil
}

func (pri *PipelineRunInterface) List(ctx context.Context, opts metav1.ListOptions) (*ExecutionSpecList, error) {
	prlist, err := pri.pipelinerunInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	rev := make(ExecutionSpecList, 0, len(prlist.Items))
	for _, pr := range prlist.Items {
		rev = append(rev, &PipelineRun{PipelineRun: &pr})
	}
	return &rev, nil
}

func (pri *PipelineRunInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (ExecutionSpec, error) {
	revPipelineRun, err := pri.pipelinerunInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun}, nil
}

type PipelineRunInformer struct {
	informer  prinformer.PipelineRunInformer
	clientset *prclientset.Clientset
	factory   prsinformers.SharedInformerFactory
}

func (pri *PipelineRunInformer) AddEventHandler(funcs cache.ResourceEventHandler) {
	pri.informer.Informer().AddEventHandler(funcs)
}

func (pri *PipelineRunInformer) HasSynced() func() bool {
	return pri.informer.Informer().HasSynced
}

func (pri *PipelineRunInformer) Get(namespace string, name string) (ExecutionSpec, bool, error) {
	pipelinerun, err := pri.informer.Lister().PipelineRuns(namespace).Get(name)
	if err != nil {
		return nil, IsNotFound(err), errors.Wrapf(err,
			"Error retrieving PipelineRun (%v) in namespace (%v): %v", name, namespace, err)
	}
	if err := pri.getStatusFromChildReferences(namespace,
		fmt.Sprintf("%s=%s", LabelKeyWorkflowRunId, pipelinerun.Labels[LabelKeyWorkflowRunId]),
		&pipelinerun.Status); err != nil {

		return nil, IsNotFound(err), errors.Wrapf(err,
			"Error retrieving the Status of the PipelineRun (%v) in namespace (%v): %v", name, namespace, err)
	}
	return NewPipelineRun(pipelinerun), false, nil
}

func (pri *PipelineRunInformer) List(labels *labels.Selector) (ExecutionSpecList, error) {
	pipelineruns, err := pri.informer.Lister().List(*labels)
	if err != nil {
		return nil, err
	}

	rev := make(ExecutionSpecList, 0, len(pipelineruns))
	for _, pipelinerun := range pipelineruns {
		rev = append(rev, NewPipelineRun(pipelinerun))
	}
	return rev, nil
}

func (pri *PipelineRunInformer) InformerFactoryStart(stopCh <-chan struct{}) {
	pri.factory.Start(stopCh)
}

func (pri *PipelineRunInformer) getStatusFromChildReferences(namespace, selector string, status *pipelineapi.PipelineRunStatus) error {
	if status.ChildReferences == nil {
		return nil
	}

	hasTaskRun, hasRun, hasCustomRun := false, false, false
	for _, child := range status.ChildReferences {
		switch child.Kind {
		case "TaskRun":
			hasTaskRun = true
		case "Run":
			hasRun = true
		case "CustomRun":
			hasCustomRun = true
		default:
		}
	}
	// TODO: restruct the workflow to contain taskrun/run status, these 2 field
	// will be removed in the future
	if hasTaskRun {
		// fetch taskrun status and insert into Status.TaskRuns
		taskruns, err := pri.clientset.TektonV1beta1().TaskRuns(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return NewInternalServerError(err, "can't fetch taskruns")
		}

		taskrunStatuses := make(map[string]*pipelineapi.PipelineRunTaskRunStatus, len(taskruns.Items))
		for _, taskrun := range taskruns.Items {
			taskrunStatuses[taskrun.Name] = &pipelineapi.PipelineRunTaskRunStatus{
				PipelineTaskName: taskrun.Labels["tekton.dev/pipelineTask"],
				Status:           taskrun.Status.DeepCopy(),
			}
		}
		status.TaskRuns = taskrunStatuses
	}
	if hasRun {
		runs, err := pri.clientset.TektonV1alpha1().Runs(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return NewInternalServerError(err, "can't fetch runs")
		}
		runStatuses := make(map[string]*pipelineapi.PipelineRunRunStatus, len(runs.Items))
		for _, run := range runs.Items {
			runStatus := run.Status.DeepCopy()
			runStatuses[run.Name] = &pipelineapi.PipelineRunRunStatus{
				PipelineTaskName: run.Labels["tekton.dev/pipelineTask"],
				Status:           FromRunStatus(runStatus),
			}
			// handle nested status
			pri.handleNestedStatus(&run, runStatus, namespace)
		}
		status.Runs = runStatuses
	}
	if hasCustomRun {
		customRuns, err := pri.clientset.TektonV1beta1().CustomRuns(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return NewInternalServerError(err, "can't fetch runs")
		}
		customRunStatuses := make(map[string]*pipelineapi.PipelineRunRunStatus, len(customRuns.Items))
		for _, customRun := range customRuns.Items {
			customRunStatus := customRun.Status.DeepCopy()
			customRunStatuses[customRun.Name] = &pipelineapi.PipelineRunRunStatus{
				PipelineTaskName: customRun.Labels["tekton.dev/pipelineTask"],
				Status:           customRunStatus,
			}
			// handle nested status
			pri.handleNestedStatusV1beta1(&customRun, customRunStatus, namespace)
		}
		if status.Runs == nil {
			status.Runs = customRunStatuses
		} else {
			for n, v := range customRunStatuses {
				status.Runs[n] = v
			}
		}
	}
	return nil
}

// handle nested status case for specific types of Run
func (pri *PipelineRunInformer) handleNestedStatus(run *pipelineapiv1alpha1.Run, runStatus *pipelineapiv1alpha1.RunStatus, namespace string) {
	var kind string
	if run.Spec.Spec != nil {
		kind = run.Spec.Spec.Kind
	} else if run.Spec.Ref != nil {
		kind = string(run.Spec.Ref.Kind)
	}
	if sort.SearchStrings(childReferencesKinds, kind) >= len(childReferencesKinds) {
		return
	}

	// need to lookup the nested status
	obj := make(map[string]interface{})
	if err := json.Unmarshal(runStatus.ExtraFields.Raw, &obj); err != nil {
		return
	}
	if pri.updateExtraFields(obj, namespace) {
		if newStatus, err := json.Marshal(obj); err == nil {
			runStatus.ExtraFields.Raw = newStatus
		}
	}

}

// handle nested status case for specific types of Run
func (pri *PipelineRunInformer) handleNestedStatusV1beta1(customRun *pipelineapi.CustomRun, customRunStatus *customRun.CustomRunStatus, namespace string) {
	var kind string
	if customRun.Spec.CustomSpec != nil {
		kind = customRun.Spec.CustomSpec.Kind
	} else if customRun.Spec.CustomRef != nil {
		kind = string(customRun.Spec.CustomRef.Kind)
	}
	if sort.SearchStrings(childReferencesKinds, kind) >= len(childReferencesKinds) {
		return
	}

	// need to lookup the nested status
	obj := make(map[string]interface{})
	if err := json.Unmarshal(customRunStatus.ExtraFields.Raw, &obj); err != nil {
		return
	}
	if pri.updateExtraFields(obj, namespace) {
		if newStatus, err := json.Marshal(obj); err == nil {
			customRunStatus.ExtraFields.Raw = newStatus
		}
	}
}

// check ExtraFields and update nested status if needed
func (pri *PipelineRunInformer) updateExtraFields(obj map[string]interface{}, namespace string) bool {
	updated := false
	val, ok := obj["pipelineRuns"]
	if !ok {
		return false
	}
	prs, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	// go through the list of pipelineRuns
	for _, val := range prs {
		probj, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		val, ok := probj["status"]
		if !ok {
			continue
		}
		statusobj, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		childRef, ok := statusobj["childReferences"]
		if !ok {
			continue
		}
		if children, ok := childRef.([]interface{}); ok {
			for _, childObj := range children {
				if child, ok := childObj.(map[string]interface{}); ok {
					kindI, ok := child["kind"]
					if !ok {
						continue
					}
					nameI, ok := child["name"]
					if !ok {
						continue
					}
					kind := fmt.Sprintf("%v", kindI)
					name := fmt.Sprintf("%v", nameI)
					if kind == "TaskRun" {
						if taskrunCR, err := pri.clientset.TektonV1beta1().TaskRuns(namespace).Get(context.Background(), name, metav1.GetOptions{}); err == nil {
							taskruns, ok := statusobj["taskRuns"]
							if !ok {
								taskruns = make(map[string]*pipelineapi.PipelineRunTaskRunStatus)
							}
							if taskrunStatus, ok := taskruns.(map[string]*pipelineapi.PipelineRunTaskRunStatus); ok {
								taskrunStatus[name] = &pipelineapi.PipelineRunTaskRunStatus{
									PipelineTaskName: taskrunCR.Labels["tekton.dev/pipelineTask"],
									Status:           taskrunCR.Status.DeepCopy(),
								}
								statusobj["taskRuns"] = taskrunStatus
								updated = true
							}
						}
					} else if kind == "Run" {
						if runCR, err := pri.clientset.TektonV1alpha1().Runs(namespace).Get(context.Background(), name, metav1.GetOptions{}); err == nil {
							runs, ok := statusobj["runs"]
							if !ok {
								runs = make(map[string]*pipelineapi.PipelineRunRunStatus)
							}
							if runStatus, ok := runs.(map[string]*pipelineapi.PipelineRunRunStatus); ok {
								runStatusStatus := runCR.Status.DeepCopy()
								runStatus[name] = &pipelineapi.PipelineRunRunStatus{
									PipelineTaskName: runCR.Labels["tekton.dev/pipelineTask"],
									Status:           FromRunStatus(runStatusStatus),
								}
								statusobj["runs"] = runStatus
								// handle nested status recursively
								pri.handleNestedStatus(runCR, runStatusStatus, namespace)
								updated = true
							}
						}
					} else if kind == "CustomRun" {
						if customRunsCR, err := pri.clientset.TektonV1beta1().CustomRuns(namespace).Get(context.Background(), name, metav1.GetOptions{}); err == nil {
							customRuns, ok := statusobj["customRuns"]
							if !ok {
								customRuns = make(map[string]*pipelineapi.PipelineRunRunStatus)
							}
							if customRunStatus, ok := customRuns.(map[string]*pipelineapi.PipelineRunRunStatus); ok {
								customRunStatusStatus := customRunsCR.Status.DeepCopy()
								customRunStatus[name] = &pipelineapi.PipelineRunRunStatus{
									PipelineTaskName: customRunsCR.Labels["tekton.dev/pipelineTask"],
									Status:           customRunStatusStatus,
								}
								statusobj["customRuns"] = customRunStatus
								// handle nested status recursively
								pri.handleNestedStatusV1beta1(customRunsCR, customRunStatusStatus, namespace)
								updated = true
							}
						}
					}
				}
			}
		}

	}
	return updated
}

// TODO: update status to v1beta1 base once pipelineloop supports v1beta1 customrun
// FromRunStatus converts a *v1alpha1.RunStatus into a corresponding *v1beta1.CustomRunStatus
func FromRunStatus(orig *v1alpha1.RunStatus) *customRun.CustomRunStatus {
	crs := customRun.CustomRunStatus{
		Status: orig.Status,
		CustomRunStatusFields: customRun.CustomRunStatusFields{
			StartTime:      orig.StartTime,
			CompletionTime: orig.CompletionTime,
			ExtraFields:    orig.ExtraFields,
		},
	}

	for _, origRes := range orig.Results {
		crs.Results = append(crs.Results, customRun.CustomRunResult{
			Name:  origRes.Name,
			Value: origRes.Value,
		})
	}

	for _, origRetryStatus := range orig.RetriesStatus {
		crs.RetriesStatus = append(crs.RetriesStatus, customRun.FromRunStatus(origRetryStatus))
	}

	return &crs
}
