// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// APIRunMetric api run metric
// swagger:model apiRunMetric
type APIRunMetric struct {

	// The display format of metric.
	Format RunMetricFormat `json:"format,omitempty"`

	// Required. The user defined name of the metric. It must between 1 and 63
	// characters long and must conform to the following regular expression:
	// `[a-z]([-a-z0-9]*[a-z0-9])?`.
	Name string `json:"name,omitempty"`

	// Required. The runtime node ID which reports the metric. The node ID can be
	// found in the RunDetail.workflow.Status. Metric with same (node_id, name)
	// are considerd as duplicate. Only the first reporting will be recorded. Max
	// length is 128.
	NodeID string `json:"node_id,omitempty"`

	// The number value of the metric.
	NumberValue float64 `json:"number_value,omitempty"`
}

// Validate validates this api run metric
func (m *APIRunMetric) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateFormat(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIRunMetric) validateFormat(formats strfmt.Registry) error {

	if swag.IsZero(m.Format) { // not required
		return nil
	}

	if err := m.Format.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("format")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIRunMetric) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIRunMetric) UnmarshalBinary(b []byte) error {
	var res APIRunMetric
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
