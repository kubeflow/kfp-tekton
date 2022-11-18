// Copyright 2018 The Kubeflow Authors
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

package resource

import (
	"context"
	"fmt"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Converted argo v1alpha1.workflow to tekton v1beta1.pipelinerun
// Rename argo fake client to tekton fake client

// Removed all the run and create tests since the Tekton client spec is constantly changing
// Removed all the k8s action tests because we will be moving to a different k8s version
// Removed all artifact spec test since Tekton doesn't use client spec to handle artifacts.
// Tests Removed: "initWithOneTimeRun", "initWithOneTimeFailedRun", "createPipeline",
// "TestCreatePipeline", "TestCreatePipeline_ComplexPipeline", "TestGetPipelineTemplate_PipelineFileNotFound",
// "TestCreateRun_ThroughPipelineID", "TestCreateRun_ThroughWorkflowSpec", "TestCreateRun_ThroughWorkflowSpecWithPatch",
// "TestCreateRun_ThroughPipelineVersion", "TestCreateRun_NoExperiment", "TestCreateRun_NullWorkflowSpec",
// "TestCreateRun_OverrideParametersError", "TestCreateRun_CreateWorkflowError", "TestCreateRun_StoreRunMetadataError",
// "TestDeleteRun", "TestDeleteRun_CrdFailure", "TestDeleteRun_DbFailure", "TestDeleteExperiment_CrdFailure",
// "TestTerminateRun", "TestTerminateRun_DbFailure", "TestRetryRun", "TestRetryRun_FailedDeletePods",
// "TestRetryRun_UpdateAndCreateFailed", "TestCreateJob_ThroughPipelineID", "TestCreateJob_ThroughPipelineVersion",
// "TestCreateJob_EmptyPipelineSpec", "TestCreateJob_InvalidWorkflowSpec", "TestCreateJob_NullWorkflowSpec",
// "TestCreateJob_ExtraInputParameterError", "TestCreateJob_FailedToCreateScheduleWorkflow", "TestEnableJob",
// "TestReportWorkflowResource_ScheduledWorkflowIDEmpty_Success", "TestReportWorkflowResource_ScheduledWorkflowIDNotEmpty_Success",
// "TestReportWorkflowResource_ScheduledWorkflowIDNotEmpty_NoExperiment_Success", "TestReportWorkflowResource_WorkflowMissingRunID",
// "TestReportWorkflowResource_WorkflowCompleted", "TestReportWorkflowResource_WorkflowCompleted_WorkflowNotFound",
// "TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted", "TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_WorkflowNotFound",
// "TestReportWorkflowResource_WorkflowCompleted_FinalStatePersisted_DeleteFailed", "TestReportScheduledWorkflowResource_Success",
// "TestReportScheduledWorkflowResource_Error", "TestGetWorkflowSpecBytes_ByWorkflowManifest", "TestGetWorkflowSpecBytes_MissingSpec",
// "TestReadArtifact_Succeed", "TestReadArtifact_WorkflowNoStatus_NotFound", "TestReadArtifact_NoRun_NotFound", "TestCreatePipelineVersion",
// "TestCreatePipelineVersion_ComplexPipelineVersion", "TestCreatePipelineVersion_CreatePipelineVersionFileError", "TestCreatePipelineVersion_GetParametersError",
// "TestCreatePipelineVersion_StorePipelineVersionMetadataError", "TestDeletePipelineVersion", "TestDeletePipelineVersion_FileError"

// Util function to create an initial state with pipeline uploaded
func initWithJobV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &api.Job{
		Name:         "j1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{PipelineManifest: v2SpecHelloWorld},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

func initEnvVars() {
	viper.Set(common.PodNamespace, "ns1")
}

func initWithOneTimeRunV2(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

type FakeBadObjectStore struct{}

func (m *FakeBadObjectStore) GetPipelineKey(pipelineID string) string {
	return pipelineID
}

func (m *FakeBadObjectStore) AddFile(template []byte, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) DeleteFile(filePath string) error {
	return errors.New("Not implemented.")
}

func (m *FakeBadObjectStore) GetFile(filePath string) ([]byte, error) {
	return []byte(""), nil
}

func (m *FakeBadObjectStore) AddAsYamlFile(o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetFromYamlFile(o interface{}, filePath string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

var testWorkflow = util.NewWorkflow(&v1beta1.PipelineRun{
	TypeMeta:   v1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "PipelineRun"},
	ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "workflow1", Namespace: "ns1"},
})

// Util function to create an initial state with pipeline uploaded
func initWithPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Pipeline) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	p, err := manager.CreatePipeline("p1", "", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, p
}

func initWithExperiment(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	return store, manager, experiment
}

func initWithExperimentAndPipeline(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	apiExperiment := &api.Experiment{Name: "e1"}
	experiment, err := manager.CreateExperiment(apiExperiment)
	assert.Nil(t, err)
	pipeline, err := manager.CreatePipeline("p1", "", "", []byte(testWorkflow.ToStringForStore()))
	assert.Nil(t, err)
	return store, manager, experiment, pipeline
}

func initWithExperimentAndPipelineAndRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.Pipeline, *model.RunDetail) {
	store, manager, exp, pipeline := initWithExperimentAndPipeline(t)
	// Create a new pipeline version with UUID being FakeUUID.
	pipelineStore, ok := store.pipelineStore.(*storage.PipelineStore)
	assert.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(FakeUUIDOne, nil))
	_, err := manager.CreatePipelineVersion(&api.PipelineVersion{
		Name: "version_for_run",
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   pipeline.UUID,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	}, []byte(testWorkflow.ToStringForStore()), true)
	assert.Nil(t, err)

	// The pipeline specified via pipeline id will be converted to this
	// pipeline's default version, which will be used to create run.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, exp, pipeline, runDetail
}

// Util function to create an initial state with pipeline uploaded
func initWithJob(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Job) {
	store, manager, exp := initWithExperiment(t)
	job := &api.Job{
		Name:         "j1",
		Enabled:      true,
		PipelineSpec: &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	j, err := manager.CreateJob(context.Background(), job)
	assert.Nil(t, err)

	return store, manager, j
}

// Removed Argo related tests (check the top page comments for more details)

func initWithPatchedRun(t *testing.T) (*FakeClientManager, *ResourceManager, *model.RunDetail) {
	store, manager, exp := initWithExperiment(t)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "{{kfp-default-bucket}}"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: exp.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	runDetail, err := manager.CreateRun(context.Background(), apiRun)
	assert.Nil(t, err)
	return store, manager, runDetail
}

// Removed Argo related tests (check the top page comments for more details)

func TestCreatePipeline_GetParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePipeline("pipeline1", "", "", []byte("I am invalid yaml"))
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "InvalidInputError")
}

func TestGetPipelineTemplate(t *testing.T) {
	store, manager, p := initWithPipeline(t)
	defer store.Close()
	actualTemplate, err := manager.GetPipelineTemplate(p.UUID)
	assert.Nil(t, err)
	assert.Equal(t, []byte(testWorkflow.ToStringForStore()), actualTemplate)
}

func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.objectStore.AddFile(template, store.objectStore.GetPipelineKey(fmt.Sprint(1)))
	manager := NewResourceManager(store)
	_, err := manager.GetPipelineTemplate("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

// Removed Argo related tests (check the top page comments for more details)

func TestCreateRun_EmptyPipelineSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to fetch manifest bytes")
}

func TestCreateRun_InvalidWorkflowSpec(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string("I am invalid"),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "world"},
			},
		},
	}
	_, err := manager.CreateRun(context.Background(), apiRun)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidInputError")
}

// Removed Argo related tests (check the top page comments for more details)

func TestDeleteRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// Removed Argo related tests (check the top page comments for more details)

func TestDeleteExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)

	_, err = manager.GetExperiment(experiment.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_ClearsDefaultExperiment(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	// Set default experiment ID. This is not normally done manually
	err := manager.SetDefaultExperimentId(experiment.UUID)
	assert.Nil(t, err)
	// Verify that default experiment ID is set
	defaultExperimentId, err := manager.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, experiment.UUID, defaultExperimentId)

	err = manager.DeleteExperiment(experiment.UUID)
	assert.Nil(t, err)

	_, err = manager.GetExperiment(experiment.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")

	// Verify that default experiment ID has been cleared
	defaultExperimentId, err = manager.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "", defaultExperimentId)
}

func TestDeleteExperiment_ExperimentNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteExperiment("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteExperiment_DbFailure(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteExperiment(experiment.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

// Removed Argo related tests (check the top page comments for more details)

func TestTerminateRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.TerminateRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// Removed Argo related tests (check the top page comments for more details)

func TestRetryRun_RunNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.RetryRun(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "not found")
}

// Removed Argo related tests (check the top page comments for more details)

func TestCreateJob_ThroughWorkflowSpec(t *testing.T) {
	store, _, job := initWithJob(t)
	defer store.Close()
	expectedJob := &model.Job{
		UUID:           "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "j1",
		Name:           "j1",
		Namespace:      "ns1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: testWorkflow.ToStringForStore(),
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "123e4567-e89b-12d3-a456-426655440000",
				ResourceType:  common.Job,
				ReferenceUUID: DefaultFakeUUID,
				ReferenceName: "e1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner,
			},
		},
	}
	assert.Equal(t, expectedJob, job)
}

// Removed Argo related tests (check the top page comments for more details)

func TestEnableJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.EnableJob(context.Background(), "1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.EnableJob(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestEnableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, v1.DeleteOptions{})
	// When swf CR is missing, enabling the job needs to fail.
	err := manager.EnableJob(context.Background(), job.UUID, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed")
	assert.Contains(t, err.Error(), "not found")
}

func TestDisableJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	require.Equal(t, job.Enabled, true)

	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, v1.DeleteOptions{})
	err := manager.EnableJob(context.Background(), job.UUID, false)
	require.Nil(t, err, "Disabling the job should succeed even when the custom resource is missing.")
	job, err = manager.GetJob(job.UUID)
	require.Nil(t, err)
	require.Equal(t, job.Enabled, false)
}

func TestEnableJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	store.DB().Close()
	err := manager.EnableJob(context.Background(), job.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteJob(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Nil(t, err)

	_, err = manager.GetJob(job.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteJob(context.Background(), "1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestDeleteJob_CustomResourceFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Delete job CR failed: some error")
}

func TestDeleteJob_CustomResourceNotFound(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	// The swf CR can be missing when user reinstalled KFP using existing DB data.
	// Explicitly delete it to simulate the situation.
	manager.getScheduledWorkflowClient(job.Namespace).Delete(context.Background(), job.Name, v1.DeleteOptions{})

	// Now deleting job should still succeed when the swf CR is already deleted.
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Nil(t, err)

	// And verify Job has been deleted from DB too.
	_, err = manager.GetJob(job.UUID)
	require.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), fmt.Sprintf("Job %v not found", job.UUID))
}

func TestDeleteJob_DbFailure(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	store.DB().Close()
	err := manager.DeleteJob(context.Background(), job.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestCreateDefaultExperiment(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	experimentID, err := manager.CreateDefaultExperiment()
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestCreateDefaultExperiment_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	experimentID, err := manager.CreateDefaultExperiment()
	assert.Nil(t, err)
	experiment, err := manager.GetExperiment(experimentID)
	assert.Nil(t, err)

	expectedExperiment := &model.Experiment{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "Default",
		Description:    "All runs created without specifying an experiment will be grouped here.",
		Namespace:      "",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	assert.Equal(t, expectedExperiment, experiment)
}

func TestGetPodLogOptions(t *testing.T) {
	initEnvVars()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := NewResourceManager(store)
	logOptions := manager.getPodLogOptions(true)
	expectedLogOptions := corev1.PodLogOptions{
		Container:  "step-main",
		Timestamps: false,
		Follow:     true,
	}
	assert.Equal(t, expectedLogOptions, logOptions)
}

var v2SpecHelloWorld = `
{
  "components": {
    "comp-hello-world": {
      "executorLabel": "exec-hello-world",
      "inputDefinitions": {
	"parameters": {
	  "text": {
	    "type": "STRING"
	  }
	}
      }
    }
  },
  "deploymentSpec": {
    "executors": {
      "exec-hello-world": {
	"container": {
	  "args": [
	    "--text",
	    "{{$.inputs.parameters['text']}}"
	  ],
	  "command": [
	    "sh",
	    "-ec",
	    "program_path=$(mktemp)\nprintf \"%s\" \"$0\" > \"$program_path\"\npython3 -u \"$program_path\" \"$@\"\n",
	    "def hello_world(text):\n    print(text)\n    return text\n\nimport argparse\n_parser = argparse.ArgumentParser(prog='Hello world', description='')\n_parser.add_argument(\"--text\", dest=\"text\", type=str, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = hello_world(**_parsed_args)\n"
	  ],
	  "image": "python:3.7"
	}
      }
    }
  },
  "pipelineInfo": {
    "name": "hello-world"
  },
  "root": {
    "dag": {
      "tasks": {
	"hello-world": {
	  "cachingOptions": {
	    "enableCache": true
	  },
	  "componentRef": {
	    "name": "comp-hello-world"
	  },
	  "inputs": {
	    "parameters": {
	      "text": {
		"componentInputParameter": "text"
	      }
	    }
	  },
	  "taskInfo": {
	    "name": "hello-world"
	  }
	}
      }
    },
    "inputDefinitions": {
      "parameters": {
	"text": {
	  "type": "STRING"
	}
      }
    }
  },
  "schemaVersion": "2.0.0",
  "sdkVersion": "kfp-1.6.5"
}
`
