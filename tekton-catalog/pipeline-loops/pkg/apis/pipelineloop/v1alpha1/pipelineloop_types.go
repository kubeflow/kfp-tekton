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

package v1alpha1

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineLoop iteratively executes a Task over elements in an array.
// +k8s:openapi-gen=true
type PipelineLoop struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the PipelineLoop from the client
	// +optional
	Spec PipelineLoopSpec `json:"spec"`
}

// PipelineLoopSpec defines the desired state of the PipelineLoop
type PipelineLoopSpec struct {
	// TaskRef is a reference to a task definition.
	// +optional
	// TaskRef     *v1beta1.TaskRef     `json:"taskRef,omitempty"`
	PipelineRef *v1beta1.PipelineRef `json:"pipelineRef,omitempty"`

	// TaskSpec is a specification of a task
	// +optional
	PipelineSpec *v1beta1.PipelineSpec `json:"pipelineSpec,omitempty"`

	// IterateParam is the name of the task parameter that is iterated upon.
	IterateParam string `json:"iterateParam"`

	// The separator for IterateParam if the IterateParam is a strings with separator char, this field is optional.
	// +optional
	IterateParamSeparator string `json:"iterateParamStringSeparator,omitempty"`

	// +optional
	IterationNumberParam string `json:"iterationNumberParam,omitempty"`

	IterateNumeric string `json:"iterateNumeric"`

	// Time after which the TaskRun times out.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Parallelism represents how many pipelines can be triggered simultaneously by the loop.
	// +optional
	Parallelism int `json:"parallelism,omitempty"`

	// Retries represents how many times a task should be retried in case of task failure.
	// +optional
	Retries int `json:"retries,omitempty"`

	// PodTemplate holds pod specific configuration
	// +optional
	PodTemplate *pod.PodTemplate `json:"podTemplate,omitempty"`

	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Workspace to a volume mapping to be consumed by a PipelineRun.
	// +optional
	Workspaces []v1beta1.WorkspaceBinding `json:"workspaces,omitempty"`

	// TaskRunSpecs holds a set of runtime specs
	// +optional
	TaskRunSpecs []v1beta1.PipelineTaskRunSpec `json:"taskRunSpecs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineLoopList contains a list of PipelineLoops
type PipelineLoopList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineLoop `json:"items"`
}

// PipelineLoopRunReason represents a reason for the Run "Succeeded" condition
type PipelineLoopRunReason string

const (
	// PipelineLoopRunReasonStarted is the reason set when the Run has just started
	PipelineLoopRunReasonStarted PipelineLoopRunReason = "Started"

	// PipelineLoopRunReasonCacheHit indicates that the Run result was fetched from cache instead of performing an actual run.
	PipelineLoopRunReasonCacheHit PipelineLoopRunReason = "CacheHit"

	// PipelineLoopRunReasonRunning indicates that the Run is in progress
	PipelineLoopRunReasonRunning PipelineLoopRunReason = "Running"

	// PipelineLoopRunReasonFailed indicates that one of the TaskRuns created from the Run failed
	PipelineLoopRunReasonFailed PipelineLoopRunReason = "Failed"

	// PipelineLoopRunReasonRecursionLimitExceeded indicates that one of the TaskRuns created from the Run has exceeded stack limit
	PipelineLoopRunReasonStackLimitExceeded PipelineLoopRunReason = "NestedCallStackLimitExceeded"

	// PipelineLoopRunReasonSucceeded indicates that all of the TaskRuns created from the Run completed successfully
	PipelineLoopRunReasonSucceeded PipelineLoopRunReason = "Succeeded"

	// PipelineLoopRunReasonCancelled indicates that a Run was cancelled.
	PipelineLoopRunReasonCancelled PipelineLoopRunReason = "PipelineLoopRunCancelled"

	// PipelineLoopRunReasonCouldntCancel indicates that a Run was cancelled but attempting to update
	// the running TaskRun as cancelled failed.
	PipelineLoopRunReasonCouldntCancel PipelineLoopRunReason = "PipelineLoopRunCouldntCancel"

	// PipelineLoopRunReasonCouldntGetPipelineLoop indicates that the associated PipelineLoop couldn't be retrieved
	PipelineLoopRunReasonCouldntGetPipelineLoop PipelineLoopRunReason = "CouldntGetPipelineLoop"

	// PipelineLoopRunReasonFailedValidation indicates that the PipelineLoop failed runtime validation
	PipelineLoopRunReasonFailedValidation PipelineLoopRunReason = "PipelineLoopValidationFailed"

	// PipelineLoopRunReasonInternalError indicates that the PipelineLoop failed due to an internal error in the reconciler
	PipelineLoopRunReasonInternalError PipelineLoopRunReason = "PipelineLoopInternalError"
)

func (t PipelineLoopRunReason) String() string {
	return string(t)
}

// PipelineLoopRunStatus contains the status stored in the ExtraFields of a Run that references a PipelineLoop.
type PipelineLoopRunStatus struct {
	// PipelineLoopSpec contains the exact spec used to instantiate the Run
	PipelineLoopSpec *PipelineLoopSpec `json:"pipelineLoopSpec,omitempty"`
	// current running pipelinerun number
	// +optional
	CurrentRunning int `json:"currentRunning,omitempty"`
	// map of PipelineLoopPipelineRunStatus with the PipelineRun name as the key
	// +optional
	PipelineRuns map[string]*PipelineLoopPipelineRunStatus `json:"pipelineRuns,omitempty"`
}

// PipelineLoopPipelineRunStatus contains the iteration number for a PipelineRun,
// current running pipeline number, and the PipelineRun's Status
type PipelineLoopPipelineRunStatus struct {
	// iteration number
	Iteration int `json:"iteration,omitempty"`
	// the current iteration item
	IterationItem interface{} `json:"iterationItem,omitempty"`
	// Status is the TaskRunStatus for the corresponding TaskRun
	// +optional
	Status *v1beta1.PipelineRunStatus `json:"status,omitempty"`
}
