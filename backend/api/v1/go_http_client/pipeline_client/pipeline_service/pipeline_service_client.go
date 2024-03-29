// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new pipeline service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for pipeline service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	PipelineServiceCreatePipeline(params *PipelineServiceCreatePipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceCreatePipelineOK, error)

	PipelineServiceCreatePipelineVersion(params *PipelineServiceCreatePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceCreatePipelineVersionOK, error)

	PipelineServiceDeletePipeline(params *PipelineServiceDeletePipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceDeletePipelineOK, error)

	PipelineServiceDeletePipelineVersion(params *PipelineServiceDeletePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceDeletePipelineVersionOK, error)

	PipelineServiceGetPipeline(params *PipelineServiceGetPipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineOK, error)

	PipelineServiceGetPipelineVersion(params *PipelineServiceGetPipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineVersionOK, error)

	PipelineServiceGetPipelineVersionTemplate(params *PipelineServiceGetPipelineVersionTemplateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineVersionTemplateOK, error)

	PipelineServiceGetTemplate(params *PipelineServiceGetTemplateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetTemplateOK, error)

	PipelineServiceListPipelineVersions(params *PipelineServiceListPipelineVersionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceListPipelineVersionsOK, error)

	PipelineServiceListPipelines(params *PipelineServiceListPipelinesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceListPipelinesOK, error)

	PipelineServiceUpdatePipelineDefaultVersion(params *PipelineServiceUpdatePipelineDefaultVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceUpdatePipelineDefaultVersionOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
PipelineServiceCreatePipeline creates a pipeline
*/
func (a *Client) PipelineServiceCreatePipeline(params *PipelineServiceCreatePipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceCreatePipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceCreatePipelineParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_CreatePipeline",
		Method:             "POST",
		PathPattern:        "/apis/v1/pipelines",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceCreatePipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceCreatePipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceCreatePipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceCreatePipelineVersion adds a pipeline version to the specified pipeline
*/
func (a *Client) PipelineServiceCreatePipelineVersion(params *PipelineServiceCreatePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceCreatePipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceCreatePipelineVersionParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_CreatePipelineVersion",
		Method:             "POST",
		PathPattern:        "/apis/v1/pipeline_versions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceCreatePipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceCreatePipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceCreatePipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceDeletePipeline deletes a pipeline and its pipeline versions
*/
func (a *Client) PipelineServiceDeletePipeline(params *PipelineServiceDeletePipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceDeletePipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceDeletePipelineParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_DeletePipeline",
		Method:             "DELETE",
		PathPattern:        "/apis/v1/pipelines/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceDeletePipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceDeletePipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceDeletePipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceDeletePipelineVersion deletes a pipeline version by pipeline version ID if the deleted pipeline version is the default pipeline version the pipeline s default version changes to the pipeline s most recent pipeline version if there are no remaining pipeline versions the pipeline will have no default version examines the run service api ipynb notebook to learn more about creating a run using a pipeline version https github com kubeflow pipelines blob master tools benchmarks run service api ipynb
*/
func (a *Client) PipelineServiceDeletePipelineVersion(params *PipelineServiceDeletePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceDeletePipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceDeletePipelineVersionParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_DeletePipelineVersion",
		Method:             "DELETE",
		PathPattern:        "/apis/v1/pipeline_versions/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceDeletePipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceDeletePipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceDeletePipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceGetPipeline finds a specific pipeline by ID
*/
func (a *Client) PipelineServiceGetPipeline(params *PipelineServiceGetPipelineParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceGetPipelineParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_GetPipeline",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipelines/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceGetPipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceGetPipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceGetPipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceGetPipelineVersion gets a pipeline version by pipeline version ID
*/
func (a *Client) PipelineServiceGetPipelineVersion(params *PipelineServiceGetPipelineVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceGetPipelineVersionParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_GetPipelineVersion",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipeline_versions/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceGetPipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceGetPipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceGetPipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceGetPipelineVersionTemplate returns a y a m l template that contains the specified pipeline version s description parameters and metadata
*/
func (a *Client) PipelineServiceGetPipelineVersionTemplate(params *PipelineServiceGetPipelineVersionTemplateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetPipelineVersionTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceGetPipelineVersionTemplateParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_GetPipelineVersionTemplate",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipeline_versions/{version_id}/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceGetPipelineVersionTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceGetPipelineVersionTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceGetPipelineVersionTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceGetTemplate returns a single y a m l template that contains the description parameters and metadata associated with the pipeline provided
*/
func (a *Client) PipelineServiceGetTemplate(params *PipelineServiceGetTemplateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceGetTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceGetTemplateParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_GetTemplate",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipelines/{id}/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceGetTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceGetTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceGetTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceListPipelineVersions lists all pipeline versions of a given pipeline
*/
func (a *Client) PipelineServiceListPipelineVersions(params *PipelineServiceListPipelineVersionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceListPipelineVersionsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceListPipelineVersionsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_ListPipelineVersions",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipeline_versions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceListPipelineVersionsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceListPipelineVersionsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceListPipelineVersionsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceListPipelines finds all pipelines
*/
func (a *Client) PipelineServiceListPipelines(params *PipelineServiceListPipelinesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceListPipelinesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceListPipelinesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_ListPipelines",
		Method:             "GET",
		PathPattern:        "/apis/v1/pipelines",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceListPipelinesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceListPipelinesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceListPipelinesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PipelineServiceUpdatePipelineDefaultVersion updates the default pipeline version of a specific pipeline
*/
func (a *Client) PipelineServiceUpdatePipelineDefaultVersion(params *PipelineServiceUpdatePipelineDefaultVersionParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PipelineServiceUpdatePipelineDefaultVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPipelineServiceUpdatePipelineDefaultVersionParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "PipelineService_UpdatePipelineDefaultVersion",
		Method:             "POST",
		PathPattern:        "/apis/v1/pipelines/{pipeline_id}/default_version/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &PipelineServiceUpdatePipelineDefaultVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PipelineServiceUpdatePipelineDefaultVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PipelineServiceUpdatePipelineDefaultVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
