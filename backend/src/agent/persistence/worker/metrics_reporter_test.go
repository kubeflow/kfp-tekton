// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReportMetrics_NoCompletedNode_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
		},
		Status: workflowapi.PipelineRunStatus{},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoRunID_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
		},
		Status: workflowapi.PipelineRunStatus{},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoArtifact_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.PipelineRunStatus{},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

func TestReportMetrics_NoMetricsArtifact_NoOP(t *testing.T) {
	pipelineFake := client.NewPipelineClientFake()

	reporter := NewMetricsReporter(pipelineFake)

	workflow := util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       types.UID("run-1"),
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.PipelineRunStatus{},
	})
	err := reporter.ReportMetrics(workflow)
	assert.Nil(t, err)
	assert.Nil(t, pipelineFake.GetReadArtifactRequest())
	assert.Nil(t, pipelineFake.GetReportedMetricsRequest())
}

// Removed TestReportMetrics_Succeed - specific to argo artifact spec

// Removed TestReportMetrics_EmptyArchive_Fail - specific to argo artifact spec

// Removed TestReportMetrics_MultipleFilesInArchive_Fail - specific to argo artifact spec

// Removed TestReportMetrics_InvalidMetricsJSON_Fail - specific to argo artifact spec

// Removed TestReportMetrics_InvalidMetricsJSON_PartialFail - specific to argo artifact spec

// Removed TestReportMetrics_CorruptedArchiveFile_Fail - specific to argo artifact spec

// Removed TestReportMetrics_MultiplMetricErrors_TransientErrowWin - specific to argo artifact spec
