// Copyright 2018 Google LLC
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
	"testing"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Replaced Argo v1alpha1.Workflow to Tekton v1beta1.PipelineRun
// Removed Argo spec test on Templates and Artifacts because Tekton API doesn't support those concepts.
// In Tekton, Artifacts are defined in a custom container where the unit tests are in the SDK.
//
// Removed tests: "TestToStringForStore", "TestWorkflow_OverrideParameters", "TestWorkflow_SetLabelsToAllTemplates",
// "TestGetWorkflowSpec", "TestGetWorkflowSpecTruncatesNameIfLongerThan200Runes", "TestVerifyParameters",
// "TestVerifyParameters_Failed", "TestFindS3ArtifactKey_Succeed", "TestFindS3ArtifactKey_ArtifactNotFound",
// "TestFindS3ArtifactKey_NodeNotFound", "TestReplaceUID", ""

func TestWorkflow_ScheduledWorkflowUUIDAsStringOrEmpty(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "MY_UID", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, true, workflow.HasScheduledWorkflowAsParent())

	// No kind
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// Wrong kind
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "WRONG_KIND",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// No API version
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ScheduledWorkflow",
				Name: "SCHEDULE_NAME",
				UID:  types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// No UID
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

}

func TestWorkflow_ScheduledAtInSecOr0(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULED_WORKFLOW_NAME",
				"scheduledworkflows.kubeflow.org/workflowEpoch":              "100",
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50"},
		},
	})
	assert.Equal(t, int64(100), workflow.ScheduledAtInSecOr0())

	// No scheduled epoch
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULED_WORKFLOW_NAME",
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50"},
		},
	})
	assert.Equal(t, int64(0), workflow.ScheduledAtInSecOr0())

	// No map
	workflow = NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})
	assert.Equal(t, int64(0), workflow.ScheduledAtInSecOr0())
}

func TestCondition(t *testing.T) {
	// No status
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		Status: workflowapi.PipelineRunStatus{},
	})
	assert.Equal(t, "", workflow.Condition())
}

// removed tests (check top page comment)

func TestWorkflow_OverrideName(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.OverrideName("NEW_WORKFLOW_NAME")

	expected := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "NEW_WORKFLOW_NAME",
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

// removed tests (check top page comment)

func TestWorkflow_SetOwnerReferences(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetOwnerReferences(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE_NAME",
		},
	})

	expected := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
				Kind:               "ScheduledWorkflow",
				Name:               "SCHEDULE_NAME",
				Controller:         BoolPointer(true),
				BlockOwnerDeletion: BoolPointer(true),
			}},
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

// removed tests (check top page comment)

func TestSetLabels(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetLabels("key", "value")

	expected := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

// removed tests (check top page comment)
