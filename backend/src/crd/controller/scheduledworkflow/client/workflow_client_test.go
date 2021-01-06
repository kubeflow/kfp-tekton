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

package client

import (
	"testing"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Removed "TestToWorkflowStatuses" because Tekton status spec is constantly changing.

func TestToWorkflowStatuses_NullOrEmpty(t *testing.T) {
	workflow := &workflowapi.PipelineRun{}

	result := toWorkflowStatuses([]*workflowapi.PipelineRun{workflow})

	expected := &swfapi.WorkflowStatus{
		Name:        "",
		Namespace:   "",
		SelfLink:    "",
		UID:         "",
		Phase:       "",
		Message:     "",
		CreatedAt:   metav1.NewTime(time.Time{}.UTC()),
		StartedAt:   metav1.NewTime(time.Time{}.UTC()),
		FinishedAt:  metav1.NewTime(time.Time{}.UTC()),
		ScheduledAt: metav1.NewTime(time.Time{}.UTC()),
		Index:       0,
	}

	assert.Equal(t, []swfapi.WorkflowStatus{*expected}, result)
}

func TestRetrieveScheduledTime(t *testing.T) {

	// Base case.
	workflow := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				commonutil.LabelKeyWorkflowEpoch: "54",
			},
		},
	}
	result := retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(54, 0).UTC()), result)

	// No label
	workflow = &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				"WRONG_LABEL": "54",
			},
		},
	}
	result = retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(50, 0).UTC()), result)

	// Parsing problem
	workflow = &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				commonutil.LabelKeyWorkflowEpoch: "UNPARSABLE_@%^%@^#%",
			},
		},
	}
	result = retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(50, 0).UTC()), result)
}

func TestRetrieveIndex(t *testing.T) {

	// Base case.
	workflow := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				commonutil.LabelKeyWorkflowIndex: "100",
			},
		},
	}
	result := retrieveIndex(workflow)
	assert.Equal(t, int64(100), result)

	// No label
	workflow = &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"WRONG_LABEL": "100",
			},
		},
	}
	result = retrieveIndex(workflow)
	assert.Equal(t, int64(0), result)

	// Parsing problem
	workflow = &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				commonutil.LabelKeyWorkflowIndex: "UNPARSABLE_LABEL_!@^^!%@#%",
			},
		},
	}
	result = retrieveIndex(workflow)
	assert.Equal(t, int64(0), result)
}

// Removed "TestLabelSelectorToGetWorkflows" because Tekton status spec is constantly changing.
