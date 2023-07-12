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

// RunServiceArchiveRunReader is a Reader for the RunServiceArchiveRun structure.
type RunServiceArchiveRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RunServiceArchiveRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRunServiceArchiveRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewRunServiceArchiveRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRunServiceArchiveRunOK creates a RunServiceArchiveRunOK with default headers values
func NewRunServiceArchiveRunOK() *RunServiceArchiveRunOK {
	return &RunServiceArchiveRunOK{}
}

/*
RunServiceArchiveRunOK describes a response with status code 200, with default header values.

A successful response.
*/
type RunServiceArchiveRunOK struct {
	Payload interface{}
}

// IsSuccess returns true when this run service archive run o k response has a 2xx status code
func (o *RunServiceArchiveRunOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this run service archive run o k response has a 3xx status code
func (o *RunServiceArchiveRunOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this run service archive run o k response has a 4xx status code
func (o *RunServiceArchiveRunOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this run service archive run o k response has a 5xx status code
func (o *RunServiceArchiveRunOK) IsServerError() bool {
	return false
}

// IsCode returns true when this run service archive run o k response a status code equal to that given
func (o *RunServiceArchiveRunOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the run service archive run o k response
func (o *RunServiceArchiveRunOK) Code() int {
	return 200
}

func (o *RunServiceArchiveRunOK) Error() string {
	return fmt.Sprintf("[POST /apis/v1/runs/{id}:archive][%d] runServiceArchiveRunOK  %+v", 200, o.Payload)
}

func (o *RunServiceArchiveRunOK) String() string {
	return fmt.Sprintf("[POST /apis/v1/runs/{id}:archive][%d] runServiceArchiveRunOK  %+v", 200, o.Payload)
}

func (o *RunServiceArchiveRunOK) GetPayload() interface{} {
	return o.Payload
}

func (o *RunServiceArchiveRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewRunServiceArchiveRunDefault creates a RunServiceArchiveRunDefault with default headers values
func NewRunServiceArchiveRunDefault(code int) *RunServiceArchiveRunDefault {
	return &RunServiceArchiveRunDefault{
		_statusCode: code,
	}
}

/*
RunServiceArchiveRunDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type RunServiceArchiveRunDefault struct {
	_statusCode int

	Payload *run_model.GooglerpcStatus
}

// IsSuccess returns true when this run service archive run default response has a 2xx status code
func (o *RunServiceArchiveRunDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this run service archive run default response has a 3xx status code
func (o *RunServiceArchiveRunDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this run service archive run default response has a 4xx status code
func (o *RunServiceArchiveRunDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this run service archive run default response has a 5xx status code
func (o *RunServiceArchiveRunDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this run service archive run default response a status code equal to that given
func (o *RunServiceArchiveRunDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the run service archive run default response
func (o *RunServiceArchiveRunDefault) Code() int {
	return o._statusCode
}

func (o *RunServiceArchiveRunDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v1/runs/{id}:archive][%d] RunService_ArchiveRun default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceArchiveRunDefault) String() string {
	return fmt.Sprintf("[POST /apis/v1/runs/{id}:archive][%d] RunService_ArchiveRun default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceArchiveRunDefault) GetPayload() *run_model.GooglerpcStatus {
	return o.Payload
}

func (o *RunServiceArchiveRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
