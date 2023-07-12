// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// V1PipelineRuntime v1 pipeline runtime
//
// swagger:model v1PipelineRuntime
type V1PipelineRuntime struct {

	// Output. The runtime JSON manifest of the pipeline, including the status
	// of pipeline steps and fields need for UI visualization etc.
	PipelineManifest string `json:"pipelineManifest,omitempty"`

	// Output. The runtime JSON manifest of the argo workflow.
	// This is deprecated after pipeline_runtime_manifest is in use.
	WorkflowManifest string `json:"workflowManifest,omitempty"`
}

// Validate validates this v1 pipeline runtime
func (m *V1PipelineRuntime) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this v1 pipeline runtime based on context it is used
func (m *V1PipelineRuntime) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *V1PipelineRuntime) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1PipelineRuntime) UnmarshalBinary(b []byte) error {
	var res V1PipelineRuntime
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
