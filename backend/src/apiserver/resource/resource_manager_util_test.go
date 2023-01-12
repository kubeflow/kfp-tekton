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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/stretchr/testify/assert"
)

func TestConvertPipelineIdToDefaultPipelineVersion(t *testing.T) {
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()
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

	// Create a run of the latest pipeline version, but by specifying the pipeline id.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
		},
	}
	expectedApiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: FakeUUIDOne},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	err = convertPipelineIdToDefaultPipelineVersion(apiRun.PipelineSpec, &apiRun.ResourceReferences, manager)
	assert.Nil(t, err)
	assert.Equal(t, expectedApiRun, apiRun)
}

// No conversion if a pipeline version already exists in resource references.
func TestConvertPipelineIdToDefaultPipelineVersion_NoOp(t *testing.T) {
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()

	// Create a new pipeline version with UUID being FakeUUID.
	oldVersionId := pipeline.DefaultVersionId
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
	// FakeUUID is the new default version's id.
	assert.NotEqual(t, oldVersionId, FakeUUIDOne)

	// Create a run by specifying both the old pipeline version and the pipeline.
	// As a result, the old version will be used and the pipeline id will be ignored.
	apiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: oldVersionId},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	expectedApiRun := &api.Run{
		Name: "run1",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: pipeline.UUID,
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID},
				Relationship: api.Relationship_OWNER,
			},
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: oldVersionId},
				Relationship: api.Relationship_CREATOR,
			},
		},
	}
	err = convertPipelineIdToDefaultPipelineVersion(apiRun.PipelineSpec, &apiRun.ResourceReferences, manager)
	assert.Nil(t, err)
	assert.Equal(t, expectedApiRun, apiRun)
}
