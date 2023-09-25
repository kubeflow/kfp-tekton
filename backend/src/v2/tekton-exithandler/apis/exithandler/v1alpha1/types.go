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

package v1alpha1

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExitHandler iteratively executes a Task over elements in an array.
type ExitHandler struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the ExitHandler from the client
	// +optional
	Spec ExitHandlerSpec `json:"spec"`
}

// ExitHandlerSpec defines the desired state of the ExitHandler
type ExitHandlerSpec struct {
	// TaskRef is a reference to a task definition.
	// +optional
	// TaskRef     *v1.TaskRef     `json:"taskRef,omitempty"`
	PipelineRef *tektonv1.PipelineRef `json:"pipelineRef,omitempty"`

	// TaskSpec is a specification of a task
	// +optional
	PipelineSpec *tektonv1.PipelineSpec `json:"pipelineSpec,omitempty"`

	// PodTemplate holds pod specific configuration
	// +optional
	PodTemplate *pod.PodTemplate `json:"podTemplate,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExitHandlerList contains a list of ExitHandlers
type ExitHandlerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExitHandler `json:"items"`
}

// ExitHandlerRunReason represents a reason for the Run "Succeeded" condition
type ExitHandlerRunReason string

const (
	// ExitHandlerRunReasonStarted is the reason set when the Run has just started
	ExitHandlerRunReasonStarted ExitHandlerRunReason = "Started"

	// ExitHandlerRunReasonCacheHit indicates that the Run result was fetched from cache instead of performing an actual run.
	ExitHandlerRunReasonCacheHit ExitHandlerRunReason = "CacheHit"

	// ExitHandlerRunReasonRunning indicates that the Run is in progress
	ExitHandlerRunReasonRunning ExitHandlerRunReason = "Running"

	// ExitHandlerRunReasonFailed indicates that one of the TaskRuns created from the Run failed
	ExitHandlerRunReasonFailed ExitHandlerRunReason = "Failed"

	// ExitHandlerRunReasonSucceeded indicates that all of the TaskRuns created from the Run completed successfully
	ExitHandlerRunReasonSucceeded ExitHandlerRunReason = "Succeeded"

	// ExitHandlerRunReasonCancelled indicates that a Run was cancelled.
	ExitHandlerRunReasonCancelled ExitHandlerRunReason = "ExitHandlerRunCancelled"

	// ExitHandlerRunReasonCouldntCancel indicates that a Run was cancelled but attempting to update
	// the running TaskRun as cancelled failed.
	ExitHandlerRunReasonCouldntCancel ExitHandlerRunReason = "ExitHandlerRunCouldntCancel"

	// ExitHandlerRunReasonCouldntGetExitHandler indicates that the associated ExitHandler couldn't be retrieved
	ExitHandlerRunReasonCouldntGetExitHandler ExitHandlerRunReason = "CouldntGetExitHandler"

	// ExitHandlerRunReasonFailedValidation indicates that the ExitHandler failed runtime validation
	ExitHandlerRunReasonFailedValidation ExitHandlerRunReason = "ExitHandlerValidationFailed"

	// ExitHandlerRunReasonInternalError indicates that the ExitHandler failed due to an internal error in the reconciler
	ExitHandlerRunReasonInternalError ExitHandlerRunReason = "ExitHandlerInternalError"
)

func (t ExitHandlerRunReason) String() string {
	return string(t)
}

// ExitHandlerRunStatus contains the status stored in the ExtraFields of a Run that references a ExitHandler.
type ExitHandlerRunStatus struct {
	// current running pipelinerun number
	// Child PipelineRun name
	ChildPipelineRun string `json:"childPipelineRun,omitempty"`
}
