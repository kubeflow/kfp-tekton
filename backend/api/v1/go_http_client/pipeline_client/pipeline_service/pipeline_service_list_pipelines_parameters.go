// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

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
	"github.com/go-openapi/swag"
)

// NewPipelineServiceListPipelinesParams creates a new PipelineServiceListPipelinesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewPipelineServiceListPipelinesParams() *PipelineServiceListPipelinesParams {
	return &PipelineServiceListPipelinesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewPipelineServiceListPipelinesParamsWithTimeout creates a new PipelineServiceListPipelinesParams object
// with the ability to set a timeout on a request.
func NewPipelineServiceListPipelinesParamsWithTimeout(timeout time.Duration) *PipelineServiceListPipelinesParams {
	return &PipelineServiceListPipelinesParams{
		timeout: timeout,
	}
}

// NewPipelineServiceListPipelinesParamsWithContext creates a new PipelineServiceListPipelinesParams object
// with the ability to set a context for a request.
func NewPipelineServiceListPipelinesParamsWithContext(ctx context.Context) *PipelineServiceListPipelinesParams {
	return &PipelineServiceListPipelinesParams{
		Context: ctx,
	}
}

// NewPipelineServiceListPipelinesParamsWithHTTPClient creates a new PipelineServiceListPipelinesParams object
// with the ability to set a custom HTTPClient for a request.
func NewPipelineServiceListPipelinesParamsWithHTTPClient(client *http.Client) *PipelineServiceListPipelinesParams {
	return &PipelineServiceListPipelinesParams{
		HTTPClient: client,
	}
}

/*
PipelineServiceListPipelinesParams contains all the parameters to send to the API endpoint

	for the pipeline service list pipelines operation.

	Typically these are written to a http.Request.
*/
type PipelineServiceListPipelinesParams struct {

	/* Filter.

	     A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1/filter.proto)).
	*/
	Filter *string

	/* PageSize.

	     The number of pipelines to be listed per page. If there are more pipelines
	than this number, the response message will contain a valid value in the
	nextPageToken field.

	     Format: int32
	*/
	PageSize *int32

	/* PageToken.

	     A page token to request the next page of results. The token is acquried
	from the nextPageToken field of the response from the previous
	ListPipelines call.
	*/
	PageToken *string

	/* ResourceReferenceKeyID.

	   The ID of the resource that referred to.
	*/
	ResourceReferenceKeyID *string

	/* ResourceReferenceKeyType.

	   The type of the resource that referred to.

	   Default: "UNKNOWN_RESOURCE_TYPE"
	*/
	ResourceReferenceKeyType *string

	/* SortBy.

	     Can be format of "field_name", "field_name asc" or "field_name desc"
	Ascending by default.
	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the pipeline service list pipelines params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PipelineServiceListPipelinesParams) WithDefaults() *PipelineServiceListPipelinesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the pipeline service list pipelines params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PipelineServiceListPipelinesParams) SetDefaults() {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)

	val := PipelineServiceListPipelinesParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithTimeout(timeout time.Duration) *PipelineServiceListPipelinesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithContext(ctx context.Context) *PipelineServiceListPipelinesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithHTTPClient(client *http.Client) *PipelineServiceListPipelinesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithFilter adds the filter to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithFilter(filter *string) *PipelineServiceListPipelinesParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithPageSize adds the pageSize to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithPageSize(pageSize *int32) *PipelineServiceListPipelinesParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithPageToken(pageToken *string) *PipelineServiceListPipelinesParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithResourceReferenceKeyID adds the resourceReferenceKeyID to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithResourceReferenceKeyID(resourceReferenceKeyID *string) *PipelineServiceListPipelinesParams {
	o.SetResourceReferenceKeyID(resourceReferenceKeyID)
	return o
}

// SetResourceReferenceKeyID adds the resourceReferenceKeyId to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetResourceReferenceKeyID(resourceReferenceKeyID *string) {
	o.ResourceReferenceKeyID = resourceReferenceKeyID
}

// WithResourceReferenceKeyType adds the resourceReferenceKeyType to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithResourceReferenceKeyType(resourceReferenceKeyType *string) *PipelineServiceListPipelinesParams {
	o.SetResourceReferenceKeyType(resourceReferenceKeyType)
	return o
}

// SetResourceReferenceKeyType adds the resourceReferenceKeyType to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetResourceReferenceKeyType(resourceReferenceKeyType *string) {
	o.ResourceReferenceKeyType = resourceReferenceKeyType
}

// WithSortBy adds the sortBy to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) WithSortBy(sortBy *string) *PipelineServiceListPipelinesParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the pipeline service list pipelines params
func (o *PipelineServiceListPipelinesParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *PipelineServiceListPipelinesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Filter != nil {

		// query param filter
		var qrFilter string

		if o.Filter != nil {
			qrFilter = *o.Filter
		}
		qFilter := qrFilter
		if qFilter != "" {

			if err := r.SetQueryParam("filter", qFilter); err != nil {
				return err
			}
		}
	}

	if o.PageSize != nil {

		// query param page_size
		var qrPageSize int32

		if o.PageSize != nil {
			qrPageSize = *o.PageSize
		}
		qPageSize := swag.FormatInt32(qrPageSize)
		if qPageSize != "" {

			if err := r.SetQueryParam("page_size", qPageSize); err != nil {
				return err
			}
		}
	}

	if o.PageToken != nil {

		// query param page_token
		var qrPageToken string

		if o.PageToken != nil {
			qrPageToken = *o.PageToken
		}
		qPageToken := qrPageToken
		if qPageToken != "" {

			if err := r.SetQueryParam("page_token", qPageToken); err != nil {
				return err
			}
		}
	}

	if o.ResourceReferenceKeyID != nil {

		// query param resource_reference_key.id
		var qrResourceReferenceKeyID string

		if o.ResourceReferenceKeyID != nil {
			qrResourceReferenceKeyID = *o.ResourceReferenceKeyID
		}
		qResourceReferenceKeyID := qrResourceReferenceKeyID
		if qResourceReferenceKeyID != "" {

			if err := r.SetQueryParam("resource_reference_key.id", qResourceReferenceKeyID); err != nil {
				return err
			}
		}
	}

	if o.ResourceReferenceKeyType != nil {

		// query param resource_reference_key.type
		var qrResourceReferenceKeyType string

		if o.ResourceReferenceKeyType != nil {
			qrResourceReferenceKeyType = *o.ResourceReferenceKeyType
		}
		qResourceReferenceKeyType := qrResourceReferenceKeyType
		if qResourceReferenceKeyType != "" {

			if err := r.SetQueryParam("resource_reference_key.type", qResourceReferenceKeyType); err != nil {
				return err
			}
		}
	}

	if o.SortBy != nil {

		// query param sort_by
		var qrSortBy string

		if o.SortBy != nil {
			qrSortBy = *o.SortBy
		}
		qSortBy := qrSortBy
		if qSortBy != "" {

			if err := r.SetQueryParam("sort_by", qSortBy); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
