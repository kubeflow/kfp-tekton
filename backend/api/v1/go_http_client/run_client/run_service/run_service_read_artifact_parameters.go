// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewRunServiceReadArtifactParams creates a new RunServiceReadArtifactParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRunServiceReadArtifactParams() *RunServiceReadArtifactParams {
	return &RunServiceReadArtifactParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRunServiceReadArtifactParamsWithTimeout creates a new RunServiceReadArtifactParams object
// with the ability to set a timeout on a request.
func NewRunServiceReadArtifactParamsWithTimeout(timeout time.Duration) *RunServiceReadArtifactParams {
	return &RunServiceReadArtifactParams{
		timeout: timeout,
	}
}

// NewRunServiceReadArtifactParamsWithContext creates a new RunServiceReadArtifactParams object
// with the ability to set a context for a request.
func NewRunServiceReadArtifactParamsWithContext(ctx context.Context) *RunServiceReadArtifactParams {
	return &RunServiceReadArtifactParams{
		Context: ctx,
	}
}

// NewRunServiceReadArtifactParamsWithHTTPClient creates a new RunServiceReadArtifactParams object
// with the ability to set a custom HTTPClient for a request.
func NewRunServiceReadArtifactParamsWithHTTPClient(client *http.Client) *RunServiceReadArtifactParams {
	return &RunServiceReadArtifactParams{
		HTTPClient: client,
	}
}

/*
RunServiceReadArtifactParams contains all the parameters to send to the API endpoint

	for the run service read artifact operation.

	Typically these are written to a http.Request.
*/
type RunServiceReadArtifactParams struct {

	/* ArtifactName.

	   The name of the artifact.
	*/
	ArtifactName string

	/* NodeID.

	   The ID of the running node.
	*/
	NodeID string

	/* RunID.

	   The ID of the run.
	*/
	RunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the run service read artifact params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RunServiceReadArtifactParams) WithDefaults() *RunServiceReadArtifactParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the run service read artifact params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RunServiceReadArtifactParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithTimeout(timeout time.Duration) *RunServiceReadArtifactParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithContext(ctx context.Context) *RunServiceReadArtifactParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithHTTPClient(client *http.Client) *RunServiceReadArtifactParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithArtifactName adds the artifactName to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithArtifactName(artifactName string) *RunServiceReadArtifactParams {
	o.SetArtifactName(artifactName)
	return o
}

// SetArtifactName adds the artifactName to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetArtifactName(artifactName string) {
	o.ArtifactName = artifactName
}

// WithNodeID adds the nodeID to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithNodeID(nodeID string) *RunServiceReadArtifactParams {
	o.SetNodeID(nodeID)
	return o
}

// SetNodeID adds the nodeId to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetNodeID(nodeID string) {
	o.NodeID = nodeID
}

// WithRunID adds the runID to the run service read artifact params
func (o *RunServiceReadArtifactParams) WithRunID(runID string) *RunServiceReadArtifactParams {
	o.SetRunID(runID)
	return o
}

// SetRunID adds the runId to the run service read artifact params
func (o *RunServiceReadArtifactParams) SetRunID(runID string) {
	o.RunID = runID
}

// WriteToRequest writes these params to a swagger request
func (o *RunServiceReadArtifactParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param artifact_name
	if err := r.SetPathParam("artifact_name", o.ArtifactName); err != nil {
		return err
	}

	// path param node_id
	if err := r.SetPathParam("node_id", o.NodeID); err != nil {
		return err
	}

	// path param run_id
	if err := r.SetPathParam("run_id", o.RunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
