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

// DeletePipelineReader is a Reader for the DeletePipeline structure.
type DeletePipelineReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeletePipelineReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewDeletePipelineOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewDeletePipelineDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeletePipelineOK creates a DeletePipelineOK with default headers values
func NewDeletePipelineOK() *DeletePipelineOK {
	return &DeletePipelineOK{}
}

/*DeletePipelineOK handles this case with default header values.

A successful response.
*/
type DeletePipelineOK struct {
	Payload interface{}
}

func (o *DeletePipelineOK) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/pipelines/{id}][%d] deletePipelineOK  %+v", 200, o.Payload)
}

func (o *DeletePipelineOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeletePipelineDefault creates a DeletePipelineDefault with default headers values
func NewDeletePipelineDefault(code int) *DeletePipelineDefault {
	return &DeletePipelineDefault{
		_statusCode: code,
	}
}

/*DeletePipelineDefault handles this case with default header values.

DeletePipelineDefault delete pipeline default
*/
type DeletePipelineDefault struct {
	_statusCode int

	Payload *pipeline_model.V1Status
}

// Code gets the status code for the delete pipeline default response
func (o *DeletePipelineDefault) Code() int {
	return o._statusCode
}

func (o *DeletePipelineDefault) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/pipelines/{id}][%d] DeletePipeline default  %+v", o._statusCode, o.Payload)
}

func (o *DeletePipelineDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.V1Status)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
