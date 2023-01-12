package server

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

// Remove argo yaml test:
// "TestCreatePipeline_YAML", "TestCreatePipeline_Tarball", "TestCreatePipeline_InvalidYAML",
// "TestCreatePipeline_InvalidURL", "TestCreatePipelineVersion_YAML", "TestCreatePipelineVersion_InvalidYAML",
// "TestCreatePipelineVersion_Tarball", "TestCreatePipelineVersion_InvalidURL"

func TestListPipelinesPublic(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}
	_, err := pipelineServer.ListPipelines(context.Background(),
		&api.ListPipelinesRequest{
			PageSize: 20,
			ResourceReferenceKey: &api.ResourceKey{
				Type: api.ResourceType_NAMESPACE,
				Id:   "",
			},
		})
	assert.EqualValues(t, nil, err, err)

}

func TestListPipelineVersion_NoResourceKey(t *testing.T) {
	httpServer := getMockServer(t)
	// Close the server when test finishes
	defer httpServer.Close()

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)

	pipelineServer := PipelineServer{resourceManager: resourceManager, httpClient: httpServer.Client(), options: &PipelineServerOptions{CollectMetrics: false}}

	_, err := pipelineServer.ListPipelineVersions(context.Background(), &api.ListPipelineVersionsRequest{
		ResourceKey: nil,
		PageSize:    20,
	})
	assert.Equal(t, "Invalid input error: ResourceKey must be set in the input", err.Error())

	//removed "pipelineServer.CreatePipeline" since the yaml spec is based on Argo yaml spec
}

func getMockServer(t *testing.T) *httptest.Server {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		file, err := os.Open("test" + req.URL.String())
		assert.Nil(t, err)
		bytes, err := ioutil.ReadAll(file)
		assert.Nil(t, err)

		rw.WriteHeader(http.StatusOK)
		rw.Write(bytes)
	}))
	return httpServer
}

func getBadMockServer() *httptest.Server {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(404)
	}))
	return httpServer
}
