// Code generated by go-swagger; DO NOT EDIT.

package job_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v1/go_http_client/job_model"
)

// JobServiceDeleteJobReader is a Reader for the JobServiceDeleteJob structure.
type JobServiceDeleteJobReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *JobServiceDeleteJobReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewJobServiceDeleteJobOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewJobServiceDeleteJobDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewJobServiceDeleteJobOK creates a JobServiceDeleteJobOK with default headers values
func NewJobServiceDeleteJobOK() *JobServiceDeleteJobOK {
	return &JobServiceDeleteJobOK{}
}

/*
JobServiceDeleteJobOK describes a response with status code 200, with default header values.

A successful response.
*/
type JobServiceDeleteJobOK struct {
	Payload interface{}
}

// IsSuccess returns true when this job service delete job o k response has a 2xx status code
func (o *JobServiceDeleteJobOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this job service delete job o k response has a 3xx status code
func (o *JobServiceDeleteJobOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this job service delete job o k response has a 4xx status code
func (o *JobServiceDeleteJobOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this job service delete job o k response has a 5xx status code
func (o *JobServiceDeleteJobOK) IsServerError() bool {
	return false
}

// IsCode returns true when this job service delete job o k response a status code equal to that given
func (o *JobServiceDeleteJobOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the job service delete job o k response
func (o *JobServiceDeleteJobOK) Code() int {
	return 200
}

func (o *JobServiceDeleteJobOK) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1/jobs/{id}][%d] jobServiceDeleteJobOK  %+v", 200, o.Payload)
}

func (o *JobServiceDeleteJobOK) String() string {
	return fmt.Sprintf("[DELETE /apis/v1/jobs/{id}][%d] jobServiceDeleteJobOK  %+v", 200, o.Payload)
}

func (o *JobServiceDeleteJobOK) GetPayload() interface{} {
	return o.Payload
}

func (o *JobServiceDeleteJobOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewJobServiceDeleteJobDefault creates a JobServiceDeleteJobDefault with default headers values
func NewJobServiceDeleteJobDefault(code int) *JobServiceDeleteJobDefault {
	return &JobServiceDeleteJobDefault{
		_statusCode: code,
	}
}

/*
JobServiceDeleteJobDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type JobServiceDeleteJobDefault struct {
	_statusCode int

	Payload *job_model.GooglerpcStatus
}

// IsSuccess returns true when this job service delete job default response has a 2xx status code
func (o *JobServiceDeleteJobDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this job service delete job default response has a 3xx status code
func (o *JobServiceDeleteJobDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this job service delete job default response has a 4xx status code
func (o *JobServiceDeleteJobDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this job service delete job default response has a 5xx status code
func (o *JobServiceDeleteJobDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this job service delete job default response a status code equal to that given
func (o *JobServiceDeleteJobDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the job service delete job default response
func (o *JobServiceDeleteJobDefault) Code() int {
	return o._statusCode
}

func (o *JobServiceDeleteJobDefault) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1/jobs/{id}][%d] JobService_DeleteJob default  %+v", o._statusCode, o.Payload)
}

func (o *JobServiceDeleteJobDefault) String() string {
	return fmt.Sprintf("[DELETE /apis/v1/jobs/{id}][%d] JobService_DeleteJob default  %+v", o._statusCode, o.Payload)
}

func (o *JobServiceDeleteJobDefault) GetPayload() *job_model.GooglerpcStatus {
	return o.Payload
}

func (o *JobServiceDeleteJobDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(job_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
