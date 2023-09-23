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
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KfpTask merges the TaskSpecPath into TaskSpec and spawns a TaskRun.
type KfpTask struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the KfpTask from the client
	Spec KfpTaskSpec `json:"spec"`
}

// KfpTaskSpec defines the desired state of the KfpTask
type KfpTaskSpec struct {
	// TaskRef is a reference to a task definition.
	// +optional
	// TaskRef     *v1beta1.TaskRef     `json:"taskRef,omitempty"`
	TaskRef *tektonv1.TaskRef `json:"taskRef,omitempty"`

	// TaskSpec is a specification of a task
	// +optional
	TaskSpec *tektonv1.TaskSpec `json:"taskSpec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KfpTaskList contains a list of KfpTasks
type KfpTaskList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KfpTask `json:"items"`
}

// KfpTaskRunReason represents a reason for the Run "Succeeded" condition
type KfpTaskRunReason string

const (
	// KfpTaskRunReasonStarted is the reason set when the Run has just started
	KfpTaskRunReasonStarted KfpTaskRunReason = "Started"

	// KfpTaskRunReasonCacheHit indicates that the Run result was fetched from cache instead of performing an actual run.
	KfpTaskRunReasonCacheHit KfpTaskRunReason = "CacheHit"

	// KfpTaskRunReasonRunning indicates that the Run is in progress
	KfpTaskRunReasonRunning KfpTaskRunReason = "Running"

	// KfpTaskRunReasonFailed indicates that one of the TaskRuns created from the Run failed
	KfpTaskRunReasonFailed KfpTaskRunReason = "Failed"

	// KfpTaskRunReasonSucceeded indicates that all of the TaskRuns created from the Run completed successfully
	KfpTaskRunReasonSucceeded KfpTaskRunReason = "Succeeded"

	// KfpTaskRunReasonCancelled indicates that a Run was cancelled.
	KfpTaskRunReasonCancelled KfpTaskRunReason = "KfpTaskRunCancelled"

	// KfpTaskRunReasonCouldntCancel indicates that a Run was cancelled but attempting to update
	// the running TaskRun as cancelled failed.
	KfpTaskRunReasonCouldntCancel KfpTaskRunReason = "KfpTaskRunCouldntCancel"

	// KfpTaskRunReasonCouldntGetKfpTask indicates that the associated KfpTask couldn't be retrieved
	KfpTaskRunReasonCouldntGetKfpTask KfpTaskRunReason = "CouldntGetKfpTask"

	// KfpTaskRunReasonFailedValidation indicates that the KfpTask failed runtime validation
	KfpTaskRunReasonFailedValidation KfpTaskRunReason = "KfpTaskalidationFailed"

	// KfpTaskRunReasonInternalError indicates that the KfpTask failed due to an internal error in the reconciler
	KfpTaskRunReasonInternalError KfpTaskRunReason = "KfpTaskInternalError"
)

func (t KfpTaskRunReason) String() string {
	return string(t)
}

// KfpTaskRunStatus contains the status stored in the ExtraFields of a Run that references a KfpTask.
type KfpTaskRunStatus struct {
	// current running TaskRun
	// Child TaskRun name
	ChildTaskRun string `json:"childTaskRun,omitempty"`
}
