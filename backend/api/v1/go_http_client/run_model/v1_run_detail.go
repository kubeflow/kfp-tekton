// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// V1RunDetail v1 run detail
//
// swagger:model v1RunDetail
type V1RunDetail struct {

	// pipeline runtime
	PipelineRuntime *V1PipelineRuntime `json:"pipelineRuntime,omitempty"`

	// run
	Run *V1Run `json:"run,omitempty"`
}

// Validate validates this v1 run detail
func (m *V1RunDetail) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePipelineRuntime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRun(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1RunDetail) validatePipelineRuntime(formats strfmt.Registry) error {
	if swag.IsZero(m.PipelineRuntime) { // not required
		return nil
	}

	if m.PipelineRuntime != nil {
		if err := m.PipelineRuntime.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pipelineRuntime")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("pipelineRuntime")
			}
			return err
		}
	}

	return nil
}

func (m *V1RunDetail) validateRun(formats strfmt.Registry) error {
	if swag.IsZero(m.Run) { // not required
		return nil
	}

	if m.Run != nil {
		if err := m.Run.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("run")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("run")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this v1 run detail based on the context it is used
func (m *V1RunDetail) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidatePipelineRuntime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateRun(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1RunDetail) contextValidatePipelineRuntime(ctx context.Context, formats strfmt.Registry) error {

	if m.PipelineRuntime != nil {
		if err := m.PipelineRuntime.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pipelineRuntime")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("pipelineRuntime")
			}
			return err
		}
	}

	return nil
}

func (m *V1RunDetail) contextValidateRun(ctx context.Context, formats strfmt.Registry) error {

	if m.Run != nil {
		if err := m.Run.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("run")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("run")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *V1RunDetail) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1RunDetail) UnmarshalBinary(b []byte) error {
	var res V1RunDetail
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
