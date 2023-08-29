package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/run_client"
	params "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

type RunInterface interface {
	Archive(params *params.RunServiceArchiveRunParams) error
	Get(params *params.RunServiceGetRunParams) (*model.V1RunDetail, *workflowapi.PipelineRun, error)
	List(params *params.RunServiceListRunsParams) ([]*model.V1Run, int, string, error)
	ListAll(params *params.RunServiceListRunsParams, maxResultSize int) ([]*model.V1Run, error)
	Unarchive(params *params.RunServiceUnarchiveRunParams) error
	Terminate(params *params.RunServiceTerminateRunParams) error
}

type RunClient struct {
	apiClient *apiclient.Run
}

func NewRunClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*RunClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &RunClient{
		apiClient: apiClient,
	}, nil
}

func (c *RunClient) Create(parameters *params.RunServiceCreateRunParams) (*model.V1RunDetail,
	*workflowapi.PipelineRun, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceCreateRun(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.RunServiceGetRunDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to create run '%v'", parameters.Run.Name))
	}

	// Unmarshal response
	var workflow workflowapi.PipelineRun
	err = yaml.Unmarshal([]byte(response.Payload.PipelineRuntime.WorkflowManifest), &workflow)
	if err != nil {
		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to unmarshal reponse. Params: %+v. Response: %s", parameters,
				response.Payload.PipelineRuntime.WorkflowManifest),
			fmt.Sprintf("Failed to unmarshal reponse"))
	}

	return response.Payload, &workflow, nil
}

func (c *RunClient) Get(parameters *params.RunServiceGetRunParams) (*model.V1RunDetail,
	*workflowapi.PipelineRun, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceGetRun(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.RunServiceGetRunDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get run '%v'", parameters.RunID))
	}

	// Unmarshal response
	var workflow workflowapi.PipelineRun
	err = yaml.Unmarshal([]byte(response.Payload.PipelineRuntime.WorkflowManifest), &workflow)
	if err != nil {
		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to unmarshal reponse. Params: %+v. Response: %s", parameters,
				response.Payload.PipelineRuntime.WorkflowManifest),
			fmt.Sprintf("Failed to unmarshal reponse"))
	}

	return response.Payload, &workflow, nil
}

func (c *RunClient) Archive(parameters *params.RunServiceArchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceArchiveRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive runs"))
	}

	return nil
}

func (c *RunClient) Unarchive(parameters *params.RunServiceUnarchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceUnarchiveRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive runs"))
	}

	return nil
}

func (c *RunClient) Delete(parameters *params.RunServiceDeleteRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceDeleteRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete runs"))
	}

	return nil
}

func (c *RunClient) List(parameters *params.RunServiceListRunsParams) (
	[]*model.V1Run, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceListRuns(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list runs"))
	}

	return response.Payload.Runs, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *RunClient) ListAll(parameters *params.RunServiceListRunsParams, maxResultSize int) (
	[]*model.V1Run, error) {
	return listAllForRun(c, parameters, maxResultSize)
}

func listAllForRun(client RunInterface, parameters *params.RunServiceListRunsParams, maxResultSize int) (
	[]*model.V1Run, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V1Run, 0)
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

func (c *RunClient) Terminate(parameters *params.RunServiceTerminateRunParams) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceTerminateRun(parameters, PassThroughAuth)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to terminate run. Params: %+v", parameters),
			fmt.Sprintf("Failed to terminate run %v", parameters.RunID))
	}
	return nil
}
