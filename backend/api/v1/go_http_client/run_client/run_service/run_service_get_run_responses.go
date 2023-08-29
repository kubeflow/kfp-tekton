// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v1/go_http_client/run_model"
)

// RunServiceGetRunReader is a Reader for the RunServiceGetRun structure.
type RunServiceGetRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RunServiceGetRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRunServiceGetRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewRunServiceGetRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRunServiceGetRunOK creates a RunServiceGetRunOK with default headers values
func NewRunServiceGetRunOK() *RunServiceGetRunOK {
	return &RunServiceGetRunOK{}
}

/*
RunServiceGetRunOK describes a response with status code 200, with default header values.

A successful response.
*/
type RunServiceGetRunOK struct {
	Payload *run_model.V1RunDetail
}

// IsSuccess returns true when this run service get run o k response has a 2xx status code
func (o *RunServiceGetRunOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this run service get run o k response has a 3xx status code
func (o *RunServiceGetRunOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this run service get run o k response has a 4xx status code
func (o *RunServiceGetRunOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this run service get run o k response has a 5xx status code
func (o *RunServiceGetRunOK) IsServerError() bool {
	return false
}

// IsCode returns true when this run service get run o k response a status code equal to that given
func (o *RunServiceGetRunOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the run service get run o k response
func (o *RunServiceGetRunOK) Code() int {
	return 200
}

func (o *RunServiceGetRunOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1/runs/{run_id}][%d] runServiceGetRunOK  %+v", 200, o.Payload)
}

func (o *RunServiceGetRunOK) String() string {
	return fmt.Sprintf("[GET /apis/v1/runs/{run_id}][%d] runServiceGetRunOK  %+v", 200, o.Payload)
}

func (o *RunServiceGetRunOK) GetPayload() *run_model.V1RunDetail {
	return o.Payload
}

func (o *RunServiceGetRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.V1RunDetail)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewRunServiceGetRunDefault creates a RunServiceGetRunDefault with default headers values
func NewRunServiceGetRunDefault(code int) *RunServiceGetRunDefault {
	return &RunServiceGetRunDefault{
		_statusCode: code,
	}
}

/*
RunServiceGetRunDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type RunServiceGetRunDefault struct {
	_statusCode int

	Payload *run_model.GooglerpcStatus
}

// IsSuccess returns true when this run service get run default response has a 2xx status code
func (o *RunServiceGetRunDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this run service get run default response has a 3xx status code
func (o *RunServiceGetRunDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this run service get run default response has a 4xx status code
func (o *RunServiceGetRunDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this run service get run default response has a 5xx status code
func (o *RunServiceGetRunDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this run service get run default response a status code equal to that given
func (o *RunServiceGetRunDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the run service get run default response
func (o *RunServiceGetRunDefault) Code() int {
	return o._statusCode
}

func (o *RunServiceGetRunDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1/runs/{run_id}][%d] RunService_GetRun default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceGetRunDefault) String() string {
	return fmt.Sprintf("[GET /apis/v1/runs/{run_id}][%d] RunService_GetRun default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceGetRunDefault) GetPayload() *run_model.GooglerpcStatus {
	return o.Payload
}

func (o *RunServiceGetRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
