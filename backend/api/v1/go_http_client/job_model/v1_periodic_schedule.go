// Code generated by go-swagger; DO NOT EDIT.

package job_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V1PeriodicSchedule PeriodicSchedule allow scheduling the job periodically with certain interval
//
// swagger:model v1PeriodicSchedule
type V1PeriodicSchedule struct {

	// The end time of the periodic job
	// Format: date-time
	EndTime strfmt.DateTime `json:"endTime,omitempty"`

	// The time interval between the starting time of consecutive jobs
	IntervalSecond string `json:"intervalSecond,omitempty"`

	// The start time of the periodic job
	// Format: date-time
	StartTime strfmt.DateTime `json:"startTime,omitempty"`
}

// Validate validates this v1 periodic schedule
func (m *V1PeriodicSchedule) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateEndTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStartTime(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1PeriodicSchedule) validateEndTime(formats strfmt.Registry) error {
	if swag.IsZero(m.EndTime) { // not required
		return nil
	}

	if err := validate.FormatOf("endTime", "body", "date-time", m.EndTime.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1PeriodicSchedule) validateStartTime(formats strfmt.Registry) error {
	if swag.IsZero(m.StartTime) { // not required
		return nil
	}

	if err := validate.FormatOf("startTime", "body", "date-time", m.StartTime.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this v1 periodic schedule based on context it is used
func (m *V1PeriodicSchedule) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *V1PeriodicSchedule) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1PeriodicSchedule) UnmarshalBinary(b []byte) error {
	var res V1PeriodicSchedule
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
