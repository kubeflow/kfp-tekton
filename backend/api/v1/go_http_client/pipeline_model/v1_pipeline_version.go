// Code generated by go-swagger; DO NOT EDIT.

package pipeline_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V1PipelineVersion v1 pipeline version
//
// swagger:model v1PipelineVersion
type V1PipelineVersion struct {

	// Input. Optional. Pipeline version code source.
	CodeSourceURL string `json:"codeSourceUrl,omitempty"`

	// Output. The time this pipeline version is created.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"createdAt,omitempty"`

	// Input. Optional. Description for the pipeline version.
	Description string `json:"description,omitempty"`

	// Output. Unique version ID. Generated by API server.
	ID string `json:"id,omitempty"`

	// Optional input field. Version name provided by user.
	Name string `json:"name,omitempty"`

	// Input. Required. Pipeline version package url.
	// Whe calling CreatePipelineVersion API method, need to provide one package
	// file location.
	PackageURL *V1URL `json:"packageUrl,omitempty"`

	// Output. The input parameters for this pipeline.
	Parameters []*V1Parameter `json:"parameters"`

	// Input field. Specify which resource this pipeline version belongs to.
	// For Experiment, the only valid resource reference is a single Namespace.
	ResourceReferences []*V1ResourceReference `json:"resourceReferences"`
}

// Validate validates this v1 pipeline version
func (m *V1PipelineVersion) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePackageURL(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateParameters(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateResourceReferences(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1PipelineVersion) validateCreatedAt(formats strfmt.Registry) error {
	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("createdAt", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1PipelineVersion) validatePackageURL(formats strfmt.Registry) error {
	if swag.IsZero(m.PackageURL) { // not required
		return nil
	}

	if m.PackageURL != nil {
		if err := m.PackageURL.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packageUrl")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("packageUrl")
			}
			return err
		}
	}

	return nil
}

func (m *V1PipelineVersion) validateParameters(formats strfmt.Registry) error {
	if swag.IsZero(m.Parameters) { // not required
		return nil
	}

	for i := 0; i < len(m.Parameters); i++ {
		if swag.IsZero(m.Parameters[i]) { // not required
			continue
		}

		if m.Parameters[i] != nil {
			if err := m.Parameters[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("parameters" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("parameters" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1PipelineVersion) validateResourceReferences(formats strfmt.Registry) error {
	if swag.IsZero(m.ResourceReferences) { // not required
		return nil
	}

	for i := 0; i < len(m.ResourceReferences); i++ {
		if swag.IsZero(m.ResourceReferences[i]) { // not required
			continue
		}

		if m.ResourceReferences[i] != nil {
			if err := m.ResourceReferences[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("resourceReferences" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("resourceReferences" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this v1 pipeline version based on the context it is used
func (m *V1PipelineVersion) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidatePackageURL(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateParameters(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateResourceReferences(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1PipelineVersion) contextValidatePackageURL(ctx context.Context, formats strfmt.Registry) error {

	if m.PackageURL != nil {
		if err := m.PackageURL.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packageUrl")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("packageUrl")
			}
			return err
		}
	}

	return nil
}

func (m *V1PipelineVersion) contextValidateParameters(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Parameters); i++ {

		if m.Parameters[i] != nil {
			if err := m.Parameters[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("parameters" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("parameters" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1PipelineVersion) contextValidateResourceReferences(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.ResourceReferences); i++ {

		if m.ResourceReferences[i] != nil {
			if err := m.ResourceReferences[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("resourceReferences" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("resourceReferences" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *V1PipelineVersion) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1PipelineVersion) UnmarshalBinary(b []byte) error {
	var res V1PipelineVersion
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
