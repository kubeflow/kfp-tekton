// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	pipeline_model "github.com/kubeflow/pipelines/backend/api/v1/go_http_client/pipeline_model"
)

// CreatePipelineVersionReader is a Reader for the CreatePipelineVersion structure.
type CreatePipelineVersionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreatePipelineVersionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewCreatePipelineVersionOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewCreatePipelineVersionDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreatePipelineVersionOK creates a CreatePipelineVersionOK with default headers values
func NewCreatePipelineVersionOK() *CreatePipelineVersionOK {
	return &CreatePipelineVersionOK{}
}

/*CreatePipelineVersionOK handles this case with default header values.

A successful response.
*/
type CreatePipelineVersionOK struct {
	Payload *pipeline_model.V1PipelineVersion
}

func (o *CreatePipelineVersionOK) Error() string {
	return fmt.Sprintf("[POST /apis/v1/pipeline_versions][%d] createPipelineVersionOK  %+v", 200, o.Payload)
}

func (o *CreatePipelineVersionOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.V1PipelineVersion)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreatePipelineVersionDefault creates a CreatePipelineVersionDefault with default headers values
func NewCreatePipelineVersionDefault(code int) *CreatePipelineVersionDefault {
	return &CreatePipelineVersionDefault{
		_statusCode: code,
	}
}

/*CreatePipelineVersionDefault handles this case with default header values.

CreatePipelineVersionDefault create pipeline version default
*/
type CreatePipelineVersionDefault struct {
	_statusCode int

	Payload *pipeline_model.V1Status
}

// Code gets the status code for the create pipeline version default response
func (o *CreatePipelineVersionDefault) Code() int {
	return o._statusCode
}

func (o *CreatePipelineVersionDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v1/pipeline_versions][%d] CreatePipelineVersion default  %+v", o._statusCode, o.Payload)
}

func (o *CreatePipelineVersionDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.V1Status)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
