package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_client"
	params "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type PipelineInterface interface {
	Create(params *params.PipelineServiceCreatePipelineParams) (*model.V1Pipeline, error)
	Get(params *params.PipelineServiceGetPipelineParams) (*model.V1Pipeline, error)
	Delete(params *params.PipelineServiceDeletePipelineParams) error
	GetTemplate(params *params.PipelineServiceGetTemplateParams) (template.Template, error)
	List(params *params.PipelineServiceListPipelinesParams) ([]*model.V1Pipeline, int, string, error)
	ListAll(params *params.PipelineServiceListPipelinesParams, maxResultSize int) (
		[]*model.V1Pipeline, error)
	UpdateDefaultVersion(params *params.PipelineServiceUpdatePipelineDefaultVersionParams) error
}

type PipelineClient struct {
	apiClient *apiclient.Pipeline
}

func (c *PipelineClient) UpdateDefaultVersion(parameters *params.PipelineServiceUpdatePipelineDefaultVersionParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()
	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.PipelineServiceUpdatePipelineDefaultVersion(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to update pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to update pipeline '%v'", parameters.PipelineID))
	}

	return nil
}

func NewPipelineClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*PipelineClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &PipelineClient{
		apiClient: apiClient,
	}, nil
}

func (c *PipelineClient) Create(parameters *params.PipelineServiceCreatePipelineParams) (*model.V1Pipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceCreatePipeline(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceCreatePipelineDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline from URL '%v'", parameters.Pipeline.URL.PipelineURL))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Get(parameters *params.PipelineServiceGetPipelineParams) (*model.V1Pipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetPipeline(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Delete(parameters *params.PipelineServiceDeletePipelineParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.PipelineServiceDeletePipeline(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceDeletePipelineDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete pipeline. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete pipeline '%v'", parameters.ID))
	}

	return nil
}

func (c *PipelineClient) GetTemplate(parameters *params.PipelineServiceGetTemplateParams) (template.Template, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetTemplate(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetTemplateDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get template. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get template for pipeline '%v'", parameters.ID))
	}

	// Unmarshal response
	return template.New([]byte(response.Payload.Template))
}

func (c *PipelineClient) List(parameters *params.PipelineServiceListPipelinesParams) (
	[]*model.V1Pipeline, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceListPipelines(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceListPipelinesDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipelines. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipelines"))
	}

	return response.Payload.Pipelines, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) ListAll(parameters *params.PipelineServiceListPipelinesParams, maxResultSize int) (
	[]*model.V1Pipeline, error) {
	return listAllForPipeline(c, parameters, maxResultSize)
}

func listAllForPipeline(client PipelineInterface, parameters *params.PipelineServiceListPipelinesParams,
	maxResultSize int) ([]*model.V1Pipeline, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V1Pipeline, 0)
	firstCall := true
	for (firstCall || (parameters.PageToken != nil && *parameters.PageToken != "")) &&
		(len(allResults) < maxResultSize) {
		results, _, pageToken, err := client.List(parameters)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
		parameters.PageToken = util.StringPointer(pageToken)
		firstCall = false
	}
	if len(allResults) > maxResultSize {
		allResults = allResults[0:maxResultSize]
	}

	return allResults, nil
}

func (c *PipelineClient) CreatePipelineVersion(parameters *params.PipelineServiceCreatePipelineVersionParams) (*model.V1PipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceCreatePipelineVersion(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceCreatePipelineVersionDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline version from URL '%v'", parameters.Version.PackageURL.PipelineURL))
	}

	return response.Payload, nil
}

func (c *PipelineClient) ListPipelineVersions(parameters *params.PipelineServiceListPipelineVersionsParams) (
	[]*model.V1PipelineVersion, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceListPipelineVersions(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceListPipelineVersionsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipeline versions. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipeline versions"))
	}

	return response.Payload.Versions, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) GetPipelineVersion(parameters *params.PipelineServiceGetPipelineVersionParams) (*model.V1PipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetPipelineVersion(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineVersionDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline version '%v'", parameters.VersionID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) GetPipelineVersionTemplate(parameters *params.PipelineServiceGetPipelineVersionTemplateParams) (
	template.Template, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetPipelineVersionTemplate(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineVersionTemplateDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get template. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get template for pipeline version '%v'", parameters.VersionID))
	}

	// Unmarshal response
	return template.New([]byte(response.Payload.Template))
}
