package server

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestListJobs_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()

	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.ListJobs(ctx, &api.ListJobsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError),
			"Failed to authorize with namespace in experiment resource reference.",
		).Error(),
	)

	_, err = server.ListJobs(ctx, &api.ListJobsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError),
			"Failed to authorize with namespace resource reference.",
		).Error(),
	)
}

var (
	commonApiJob = &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Relationship: api.Relationship_OWNER,
			},
		},
	}

	commonExpectedJob = &api.Job{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "job1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt: &timestamp.Timestamp{Seconds: 2},
		UpdatedAt: &timestamp.Timestamp{Seconds: 2},
		Status:    "NO_STATUS",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Name: "exp1", Relationship: api.Relationship_OWNER,
			},
		},
	}
)

func TestValidateApiJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_ValidateNoExperimentResourceReferenceSucceeds(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		// This job has no ResourceReferences, no experiment
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithInvalidPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: referencesOfExperimentAndInvalidPipelineVersion,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestValidateApiJob_NoValidPipelineSpecOrPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReference,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidateApiJob_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateApiJob_ValidatePipelineSpecFailed(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			PipelineId: "not_exist_pipeline",
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Pipeline not_exist_pipeline not found")
}

func TestValidateApiJob_InvalidCron(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * ",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Schedule cron is not a supported format")
}

// remove argo spec test:
// "TestValidateApiJob_MaxConcurrencyOutOfRange", "TestValidateApiJob_NegativeIntervalSecond", "TestCreateJob", "TestCreateJob_Unauthorized",
// "TestGetJob_Unauthorized", "TestGetJob_Multiuser", "TestListJobs_Unauthorized", "TestListJobs_Multiuser", "TestEnableJob_Unauthorized",
// "TestEnableJob_Multiuser", "TestDisableJob_Unauthorized", "TestDisableJob_Multiuser", ""
