// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	run_model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
)

// ArchiveRunReader is a Reader for the ArchiveRun structure.
type ArchiveRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ArchiveRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewArchiveRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewArchiveRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewArchiveRunOK creates a ArchiveRunOK with default headers values
func NewArchiveRunOK() *ArchiveRunOK {
	return &ArchiveRunOK{}
}

/*ArchiveRunOK handles this case with default header values.

A successful response.
*/
type ArchiveRunOK struct {
	Payload interface{}
}

func (o *ArchiveRunOK) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/runs/{id}:archive][%d] archiveRunOK  %+v", 200, o.Payload)
}

func (o *ArchiveRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewArchiveRunDefault creates a ArchiveRunDefault with default headers values
func NewArchiveRunDefault(code int) *ArchiveRunDefault {
	return &ArchiveRunDefault{
		_statusCode: code,
	}
}

/*ArchiveRunDefault handles this case with default header values.

ArchiveRunDefault archive run default
*/
type ArchiveRunDefault struct {
	_statusCode int

	Payload *run_model.V1beta1Status
}

// Code gets the status code for the archive run default response
func (o *ArchiveRunDefault) Code() int {
	return o._statusCode
}

func (o *ArchiveRunDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/runs/{id}:archive][%d] ArchiveRun default  %+v", o._statusCode, o.Payload)
}

func (o *ArchiveRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.V1beta1Status)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
