// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new experiment service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for experiment service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
ExperimentServiceArchiveExperimentV1 archives an experiment and the experiment s runs and jobs
*/
func (a *Client) ExperimentServiceArchiveExperimentV1(params *ExperimentServiceArchiveExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceArchiveExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceArchiveExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_ArchiveExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments/{id}:archive",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceArchiveExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceArchiveExperimentV1OK), nil

}

/*
ExperimentServiceCreateExperimentV1 creates a new experiment
*/
func (a *Client) ExperimentServiceCreateExperimentV1(params *ExperimentServiceCreateExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceCreateExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceCreateExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_CreateExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceCreateExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceCreateExperimentV1OK), nil

}

/*
ExperimentServiceDeleteExperimentV1 deletes an experiment without deleting the experiment s runs and jobs to avoid unexpected behaviors delete an experiment s runs and jobs before deleting the experiment
*/
func (a *Client) ExperimentServiceDeleteExperimentV1(params *ExperimentServiceDeleteExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceDeleteExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceDeleteExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_DeleteExperimentV1",
		Method:             "DELETE",
		PathPattern:        "/apis/v1beta1/experiments/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceDeleteExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceDeleteExperimentV1OK), nil

}

/*
ExperimentServiceGetExperimentV1 finds a specific experiment by ID
*/
func (a *Client) ExperimentServiceGetExperimentV1(params *ExperimentServiceGetExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceGetExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceGetExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_GetExperimentV1",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/experiments/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceGetExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceGetExperimentV1OK), nil

}

/*
ExperimentServiceListExperimentsV1 finds all experiments supports pagination and sorting on certain fields
*/
func (a *Client) ExperimentServiceListExperimentsV1(params *ExperimentServiceListExperimentsV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceListExperimentsV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceListExperimentsV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_ListExperimentsV1",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/experiments",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceListExperimentsV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceListExperimentsV1OK), nil

}

/*
ExperimentServiceUnarchiveExperimentV1 restores an archived experiment the experiment s archived runs and jobs will stay archived
*/
func (a *Client) ExperimentServiceUnarchiveExperimentV1(params *ExperimentServiceUnarchiveExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ExperimentServiceUnarchiveExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewExperimentServiceUnarchiveExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ExperimentService_UnarchiveExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments/{id}:unarchive",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ExperimentServiceUnarchiveExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ExperimentServiceUnarchiveExperimentV1OK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
