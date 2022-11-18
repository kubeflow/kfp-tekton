// Code generated by go-swagger; DO NOT EDIT.

package job_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	job_model "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/job_model"
)

// DeleteJobReader is a Reader for the DeleteJob structure.
type DeleteJobReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteJobReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewDeleteJobOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewDeleteJobDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteJobOK creates a DeleteJobOK with default headers values
func NewDeleteJobOK() *DeleteJobOK {
	return &DeleteJobOK{}
}

/*DeleteJobOK handles this case with default header values.

A successful response.
*/
type DeleteJobOK struct {
	Payload interface{}
}

func (o *DeleteJobOK) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/jobs/{id}][%d] deleteJobOK  %+v", 200, o.Payload)
}

func (o *DeleteJobOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteJobDefault creates a DeleteJobDefault with default headers values
func NewDeleteJobDefault(code int) *DeleteJobDefault {
	return &DeleteJobDefault{
		_statusCode: code,
	}
}

/*DeleteJobDefault handles this case with default header values.

DeleteJobDefault delete job default
*/
type DeleteJobDefault struct {
	_statusCode int

	Payload *job_model.V1Status
}

// Code gets the status code for the delete job default response
func (o *DeleteJobDefault) Code() int {
	return o._statusCode
}

func (o *DeleteJobDefault) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/jobs/{id}][%d] DeleteJob default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteJobDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(job_model.V1Status)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
