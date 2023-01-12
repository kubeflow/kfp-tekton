package server

import (
	"encoding/json"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Converted argo v1alpha1.workflow to tekton v1beta1.pipelinerun
// removed tests: "TestReportWorkflow"

func TestReportWorkflow_ValidationFailed(t *testing.T) {
	clientManager, resourceManager, run := initWithOneTimeRun(t)
	defer clientManager.Close()
	reportServer := NewReportServer(resourceManager)

	workflow := util.NewWorkflow(&v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			UID:       types.UID(run.UUID),
		},
	})

	_, err := reportServer.ReportWorkflow(nil, &api.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestValidateReportWorkflowRequest(t *testing.T) {
	// Name
	workflow := &workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	}
	marshalledWorkflow, _ := json.Marshal(workflow)
	generatedWorkflow, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: string(marshalledWorkflow)})
	assert.Nil(t, err)
	assert.Equal(t, *util.NewWorkflow(workflow), *generatedWorkflow)
}

func TestValidateReportWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: "WRONG WORKFLOW"})
	assert.NotNil(t, err)
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportWorkflowRequest_MissingField(t *testing.T) {
	// Name
	workflow := util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	_, err := ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	workflow = util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})

	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	workflow = util.NewWorkflow(&workflowapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})

	_, err = ValidateReportWorkflowRequest(&api.ReportWorkflowRequest{Workflow: workflow.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The workflow must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}

func TestValidateReportScheduledWorkflowRequest_UnmarshalError(t *testing.T) {
	_, err := ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: "WRONG_SCHEDULED_WORKFLOW"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestValidateReportScheduledWorkflowRequest_MissingField(t *testing.T) {
	// Name
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			UID:       "1",
		},
	})

	_, err := ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a name")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// Namespace
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_NAME",
			UID:  "1",
		},
	})

	_, err = ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a namespace")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)

	// UID
	swf = util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
		},
	})

	_, err = ValidateReportScheduledWorkflowRequest(
		&api.ReportScheduledWorkflowRequest{ScheduledWorkflow: swf.ToStringForStore()})
	assert.NotNil(t, err)
	assert.Contains(t, err.(*util.UserError).ExternalMessage(), "The resource must have a UID")
	assert.Equal(t, err.(*util.UserError).ExternalStatusCode(), codes.InvalidArgument)
}
