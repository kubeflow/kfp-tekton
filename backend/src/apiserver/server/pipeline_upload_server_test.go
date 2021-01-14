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

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

const (
	fakeVersionUUID = "123e4567-e89b-12d3-a456-526655440000"
	fakeVersionName = "a_fake_version_name"
)

func TestUploadPipeline_YAML(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := PipelineUploadServer{resourceManager: resourceManager}
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	part, _ := w.CreateFormFile("uploadfile", "hello-world.yaml")
	io.Copy(part, bytes.NewBufferString("apiVersion: tekton.dev/v1beta1\nkind: PipelineRun"))
	w.Close()
	req, _ := http.NewRequest("POST", "/apis/v1beta1/pipelines/upload", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.UploadPipeline)
	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code)
	// Verify time format is RFC3339.
	parsedResponse := struct {
		CreatedAt      string `json:"created_at"`
		DefaultVersion struct {
			CreatedAt string `json:"created_at"`
		} `json:"default_version"`
	}{}
	json.Unmarshal(response.Body.Bytes(), &parsedResponse)
	assert.Equal(t, "1970-01-01T00:00:01Z", parsedResponse.CreatedAt)
	assert.Equal(t, "1970-01-01T00:00:01Z", parsedResponse.DefaultVersion.CreatedAt)

	// Verify stored in object store
	objStore := clientManager.ObjectStore()
	template, err := objStore.GetFile(objStore.GetPipelineKey(resource.DefaultFakeUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)

	opts, err := list.NewOptions(&model.Pipeline{}, 2, "", nil)
	assert.Nil(t, err)

	// Verify metadata in db
	pkgsExpect := []*model.Pipeline{
		{
			UUID:             resource.DefaultFakeUUID,
			CreatedAtInSec:   1,
			Name:             "hello-world.yaml",
			Parameters:       "[]",
			Status:           model.PipelineReady,
			DefaultVersionId: resource.DefaultFakeUUID,
			DefaultVersion: &model.PipelineVersion{
				UUID:           resource.DefaultFakeUUID,
				CreatedAtInSec: 1,
				Name:           "hello-world.yaml",
				Parameters:     "[]",
				Status:         model.PipelineVersionReady,
				PipelineId:     resource.DefaultFakeUUID,
			}}}
	pkg, totalSize, str, err := clientManager.PipelineStore().ListPipelines(opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, pkgsExpect, pkg)

	// Upload a new version under this pipeline

	// Set the fake uuid generator with a new uuid to avoid generate a same uuid as above.
	server = updateClientManager(clientManager, util.NewFakeUUIDGeneratorOrFatal(fakeVersionUUID, nil))
	response = uploadPipeline("/apis/v1beta1/pipelines/upload_version?name="+fakeVersionName+"&pipelineid="+resource.DefaultFakeUUID,
		bytes.NewReader(bytesBuffer.Bytes()), writer, server.UploadPipelineVersion)
	assert.Equal(t, 200, response.Code)
	assert.Contains(t, response.Body.String(), `"created_at":"1970-01-01T00:00:02Z"`)

	// Verify stored in object store
	objStore = clientManager.ObjectStore()
	template, err = objStore.GetFile(objStore.GetPipelineKey(fakeVersionUUID))
	assert.Nil(t, err)
	assert.NotNil(t, template)
	opts, err = list.NewOptions(&model.PipelineVersion{}, 2, "", nil)
	assert.Nil(t, err)

	// Verify metadata in db
	versionsExpect := []*model.PipelineVersion{
		{
			UUID:           resource.DefaultFakeUUID,
			CreatedAtInSec: 1,
			Name:           "hello-world.yaml",
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
		{
			UUID:           fakeVersionUUID,
			CreatedAtInSec: 2,
			Name:           fakeVersionName,
			Parameters:     "[]",
			Status:         model.PipelineVersionReady,
			PipelineId:     resource.DefaultFakeUUID,
		},
	}
	// Expect 2 versions, one is created by default when creating pipeline and the other is what we manually created
	versions, total_size, str, err := clientManager.PipelineStore().ListPipelineVersions(resource.DefaultFakeUUID, opts)
	assert.Nil(t, err)
	assert.Equal(t, str, "")
	assert.Equal(t, 2, total_size)
	assert.Equal(t, versionsExpect, versions)
}

// Removed TestUploadPipeline tests because it expects argo spec in the tarball file.
// Need to update the tarball spec once we finalized the Tekton custom multi-task loop client.
// Tests removed: "TestUploadPipeline_Tarball", "TestUploadPipeline_GetFormFileError", "TestUploadPipeline_SpecifyFileName",
// "TestUploadPipeline_FileNameTooLong", "TestUploadPipeline_SpecifyFileDescription", "TestUploadPipelineVersion_GetFromFileError",
// "TestUploadPipelineVersion_FileNameTooLong"
