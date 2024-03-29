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

package storage

import (
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	defaultFakePipelineId      = "123e4567-e89b-12d3-a456-426655440000"
	defaultFakePipelineIdTwo   = "123e4567-e89b-12d3-a456-426655440001"
	defaultFakePipelineIdThree = "123e4567-e89b-12d3-a456-426655440002"
	defaultFakePipelineIdFour  = "123e4567-e89b-12d3-a456-426655440003"
	defaultFakePipelineIdFive  = "123e4567-e89b-12d3-a456-426655440004"
)

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{
		Name:       name,
		Parameters: `[{"Name": "param1"}]`,
		Status:     model.PipelineReady,
		DefaultVersion: &model.PipelineVersion{
			Name:       name,
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineVersionReady,
		}}
}

func TestListPipelines_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(&model.Pipeline{
		Name:   "pipeline3",
		Status: model.PipelineCreating,
		DefaultVersion: &model.PipelineVersion{
			Name:   "pipeline3",
			Status: model.PipelineVersionCreating}})

	expectedPipeline1 := &model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline2 := &model.Pipeline{
		UUID:             defaultFakePipelineIdTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdTwo,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_WithFilter(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline_foo"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline_bar"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)

	expectedPipeline1 := &model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline_foo",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline_foo",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	filterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_f"},
			},
		},
	}
	opts, err := list.NewOptions(&model.Pipeline{}, 10, "id", filterProto)
	assert.Nil(t, err)

	pipelines, totalSize, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelines_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))
	expectedPipeline1 := &model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline4 := &model.Pipeline{
		UUID:             defaultFakePipelineIdFour,
		CreatedAtInSec:   4,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdFour,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdFour,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1, expectedPipeline4}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline2 := &model.Pipeline{
		UUID:             defaultFakePipelineIdTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline3",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdTwo,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline3 := &model.Pipeline{
		UUID:             defaultFakePipelineIdThree,
		CreatedAtInSec:   3,
		Name:             "pipeline4",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdThree,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdThree,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline2, expectedPipeline3}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)

	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline3"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline4"))
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipeline(createPipeline("pipeline2"))

	expectedPipeline2 := &model.Pipeline{
		UUID:             defaultFakePipelineIdTwo,
		CreatedAtInSec:   2,
		Name:             "pipeline3",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdTwo,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdTwo,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline3 := &model.Pipeline{
		UUID:             defaultFakePipelineIdThree,
		CreatedAtInSec:   3,
		Name:             "pipeline4",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdThree,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdThree,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline3, expectedPipeline2}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)

	expectedPipeline1 := &model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}
	expectedPipeline4 := &model.Pipeline{
		UUID:             defaultFakePipelineIdFour,
		CreatedAtInSec:   4,
		Name:             "pipeline2",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineIdFour,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineIdFour,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected2 := []*model.Pipeline{expectedPipeline4, expectedPipeline1}

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err = pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelinesExpected2, pipelines)
}

func TestListPipelines_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	expectedPipeline1 := &model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}
	pipelinesExpected := []*model.Pipeline{expectedPipeline1}

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	pipelines, total_size, nextPageToken, err := pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelinesExpected, pipelines)
}

func TestListPipelinesError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelines(&common.FilterContext{}, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	pipelineExpected := model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		}}

	pipeline, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:   "pipeline3",
			Status: model.PipelineCreating,
			DefaultVersion: &model.PipelineVersion{
				Name:   "pipeline3",
				Status: model.PipelineVersionCreating,
			}})

	_, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipeline_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	_, err := pipelineStore.GetPipeline("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestCreatePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineExpected := model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineReady,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			Status:         model.PipelineVersionReady,
			PipelineId:     defaultFakePipelineId,
		}}

	pipeline := createPipeline("pipeline1")
	pipeline, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline, "Got unexpected pipeline.")
}

func TestCreatePipeline_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	pipeline := createPipeline("pipeline1")
	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	_, err = pipelineStore.CreatePipeline(pipeline)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline1 already exist")
}

func TestCreatePipeline_InternalServerError(t *testing.T) {
	pipeline := &model.Pipeline{
		Name:           "Pipeline123",
		DefaultVersion: &model.PipelineVersion{}}
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()

	_, err := pipelineStore.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestDeletePipeline(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	err := pipelineStore.DeletePipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	_, err = pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipelineError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.DeletePipeline(defaultFakePipelineId)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	pipeline, err := pipelineStore.CreatePipeline(createPipeline("pipeline1"))
	assert.Nil(t, err)
	pipelineExpected := model.Pipeline{
		UUID:             defaultFakePipelineId,
		CreatedAtInSec:   1,
		Name:             "pipeline1",
		Parameters:       `[{"Name": "param1"}]`,
		Status:           model.PipelineDeleting,
		DefaultVersionId: defaultFakePipelineId,
		DefaultVersion: &model.PipelineVersion{
			UUID:           defaultFakePipelineId,
			CreatedAtInSec: 1,
			Name:           "pipeline1",
			Parameters:     `[{"Name": "param1"}]`,
			Status:         model.PipelineVersionDeleting,
			PipelineId:     defaultFakePipelineId,
		},
	}
	err = pipelineStore.UpdatePipelineStatus(pipeline.UUID, model.PipelineDeleting)
	assert.Nil(t, err)
	err = pipelineStore.UpdatePipelineVersionStatus(pipeline.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	pipeline, err = pipelineStore.GetPipelineWithStatus(defaultFakePipelineId, model.PipelineDeleting)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, *pipeline)
}

func TestUpdatePipelineStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))
	db.Close()
	err := pipelineStore.UpdatePipelineStatus(defaultFakePipelineId, model.PipelineDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestCreatePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		Description:   "pipeline_version_description",
		PipelineId:    defaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion, true)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Description:    "pipeline_version_description",
		Status:         model.PipelineVersionCreating,
		PipelineId:     defaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")

	// Check whether pipeline has updated default version id.
	pipeline, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, defaultFakePipelineIdTwo, "Got unexpected default version id.")
	assert.Equal(t, pipeline.DefaultVersion.Description, "pipeline_version_description", "Got unexpected description for pipeline version.")
}

func TestCreatePipelineVersionNotUpdateDefaultVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline first.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion := &model.PipelineVersion{
		Name:          "pipeline_version_1",
		Parameters:    `[{"Name": "param1"}]`,
		PipelineId:    defaultFakePipelineId,
		Status:        model.PipelineVersionCreating,
		CodeSourceUrl: "code_source_url",
	}
	pipelineVersionCreated, err := pipelineStore.CreatePipelineVersion(
		pipelineVersion, false)

	// Check whether created pipeline version is as expected.
	pipelineVersionExpected := model.PipelineVersion{
		UUID:           defaultFakePipelineIdTwo,
		CreatedAtInSec: 2,
		Name:           "pipeline_version_1",
		Parameters:     `[{"Name": "param1"}]`,
		Status:         model.PipelineVersionCreating,
		PipelineId:     defaultFakePipelineId,
		CodeSourceUrl:  "code_source_url",
	}
	assert.Nil(t, err)
	assert.Equal(
		t,
		pipelineVersionExpected,
		*pipelineVersionCreated,
		"Got unexpected pipeline.")

	// Check whether pipeline has updated default version id.
	pipeline, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.NotEqual(t, pipeline.DefaultVersionId, defaultFakePipelineIdTwo, "Got unexpected default version id.")
	assert.Equal(t, pipeline.DefaultVersionId, defaultFakePipelineId, "Got unexpected default version id.")

}

func TestCreatePipelineVersion_DuplicateKey(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		}, true)

	// Create another new version with same name.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param2"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		}, true)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The name pipeline_version_1 already exist")
}

func TestCreatePipelineVersion_InternalServerError_DBClosed(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Try to create a new version but db is closed.
	_, err := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
		}, true)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline version to return error")
}

func TestDeletePipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a second version, which will become the default version.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Delete version with id being defaultFakePipelineIdThree.
	err := pipelineStore.DeletePipelineVersion(defaultFakePipelineIdThree)
	assert.Nil(t, err)

	// Check version removed.
	_, err = pipelineStore.GetPipelineVersion(defaultFakePipelineIdThree)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())

	// Check new default version is version with id being defaultFakePipelineIdTwo.
	pipeline, err := pipelineStore.GetPipeline(defaultFakePipelineId)
	assert.Nil(t, err)
	assert.Equal(t, pipeline.DefaultVersionId, defaultFakePipelineIdTwo)
}

func TestDeletePipelineVersionError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	db.Close()
	// On closed db, create pipeline version ends in internal error.
	err := pipelineStore.DeletePipelineVersion(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineVersion(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Get pipeline version.
	pipelineVersion, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Nil(t, err)
	assert.Equal(
		t,
		model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		*pipelineVersion, "Got unexpected pipeline version.")
}

func TestGetPipelineVersion_InternalError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	_, err := pipelineStore.GetPipelineVersion("123")
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestGetPipelineVersion_NotFound_VersionStatusCreating(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		}, true)

	_, err := pipelineStore.GetPipelineVersion(defaultFakePipelineIdTwo)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestGetPipelineVersion_NotFoundError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	_, err := pipelineStore.GetPipelineVersion(defaultFakePipelineId)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found")
}

func TestListPipelineVersion_FilterOutNotReady(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a first version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a second version with status ready.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create a third version with status creating.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionCreating,
		}, true)

	pipelineVersionsExpected := []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady}}

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)

	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)

	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, total_size)
	assert.Equal(t, pipelineVersionsExpected, pipelineVersions)
}

func TestListPipelineVersions_Pagination(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_3" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with defaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_4" with defaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// List results in 2 pages: first page containing version_1 and version_2;
	// and second page containing verion_3 and version_4.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)

	// Second page.
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_Pagination_Descend(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:        "pipeline_version_1",
			Parameters:  `[{"Name": "param1"}]`,
			PipelineId:  defaultFakePipelineId,
			Status:      model.PipelineVersionReady,
			Description: "version_1",
		}, true)

	// Create "version_3" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_3",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with defaultFakePipelineIdFour.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFour, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_4" with defaultFakePipelineIdFive.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdFive, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_4",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// List result in 2 pages: first page "version_4" and "version_3"; second
	// page "version_2" and "version_1".
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "name desc", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.NotEmpty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// First page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFive,
			CreatedAtInSec: 5,
			Name:           "pipeline_version_4",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdThree,
			CreatedAtInSec: 3,
			Name:           "pipeline_version_3",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})

	opts, err = list.NewOptionsFromToken(nextPageToken, 2)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err =
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Empty(t, nextPageToken)
	assert.Equal(t, 4, total_size)

	// Second Page.
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdFour,
			CreatedAtInSec: 4,
			Name:           "pipeline_version_2",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			CreatedAtInSec: 2,
			Name:           "pipeline_version_1",
			Description:    "version_1",
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_Pagination_LessThanPageSize(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	pipelineVersions, total_size, nextPageToken, err :=
		pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, total_size)
	assert.Equal(t, pipelineVersions, []*model.PipelineVersion{
		&model.PipelineVersion{
			UUID:           defaultFakePipelineIdTwo,
			Name:           "pipeline_version_1",
			CreatedAtInSec: 2,
			Parameters:     `[{"Name": "param1"}]`,
			PipelineId:     defaultFakePipelineId,
			Status:         model.PipelineVersionReady,
		},
	})
}

func TestListPipelineVersions_WithFilter(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create "version_1" with defaultFakePipelineIdTwo.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Create "version_2" with defaultFakePipelineIdThree.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdThree, nil)
	pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_2",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Filter for name being equal to pipeline_version_1
	equalFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version_1"},
			},
		},
	}

	// Filter for name prefix being pipeline_version
	prefixFilterProto := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_IS_SUBSTRING,
				Value: &api.Predicate_StringValue{StringValue: "pipeline_version"},
			},
		},
	}

	// Only return 1 pipeline version with equal filter.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", equalFilterProto)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err := pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 1, totalSize)

	// Return 2 pipeline versions without filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)

	// Return 2 pipeline versions with prefix filter.
	opts, err = list.NewOptions(&model.PipelineVersion{}, 10, "id", prefixFilterProto)
	assert.Nil(t, err)
	_, totalSize, nextPageToken, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Nil(t, err)
	assert.Equal(t, "", nextPageToken)
	assert.Equal(t, 2, totalSize)
}

func TestListPipelineVersionsError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)
	_, _, _, err = pipelineStore.ListPipelineVersions(defaultFakePipelineId, opts)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}

func TestUpdatePipelineVersionStatus(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	// Create a pipeline.
	pipelineStore.CreatePipeline(
		&model.Pipeline{
			Name:       "pipeline_1",
			Parameters: `[{"Name": "param1"}]`,
			Status:     model.PipelineReady,
		})

	// Create a version under the above pipeline.
	pipelineStore.uuid = util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineIdTwo, nil)
	pipelineVersion, _ := pipelineStore.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:       "pipeline_version_1",
			Parameters: `[{"Name": "param1"}]`,
			PipelineId: defaultFakePipelineId,
			Status:     model.PipelineVersionReady,
		}, true)

	// Change version to deleting status
	err := pipelineStore.UpdatePipelineVersionStatus(
		pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)

	// Check the new status by retrieving this pipeline version.
	retrievedPipelineVersion, err :=
		pipelineStore.GetPipelineVersionWithStatus(
			pipelineVersion.UUID, model.PipelineVersionDeleting)
	assert.Nil(t, err)
	assert.Equal(t, *retrievedPipelineVersion, model.PipelineVersion{
		UUID:           defaultFakePipelineIdTwo,
		Name:           "pipeline_version_1",
		CreatedAtInSec: 2,
		Parameters:     `[{"Name": "param1"}]`,
		PipelineId:     defaultFakePipelineId,
		Status:         model.PipelineVersionDeleting,
	})
}

func TestUpdatePipelineVersionStatusError(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	pipelineStore := NewPipelineStore(
		db,
		util.NewFakeTimeForEpoch(),
		util.NewFakeUUIDGeneratorOrFatal(defaultFakePipelineId, nil))

	db.Close()
	// Internal error because of closed DB.
	err := pipelineStore.UpdatePipelineVersionStatus(
		defaultFakePipelineId, model.PipelineVersionDeleting)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
}
