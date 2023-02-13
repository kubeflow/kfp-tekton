/*
// Copyright 2022 kubeflow.org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/

package v1alpha1

import (
	"context"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*ExitHandler)(nil)

// Validate ExitHandler
func (eh *ExitHandler) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(eh.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return eh.Spec.Validate(ctx)
}

// Validate ExitHandlerSpec
func (ehs *ExitHandlerSpec) Validate(ctx context.Context) *apis.FieldError {
	// Validate Task reference or inline task spec.
	if err := validatePipelineSpec(ctx, ehs); err != nil {
		return err
	}
	return nil
}

func validatePipelineSpec(ctx context.Context, ehs *ExitHandlerSpec) *apis.FieldError {
	// pipelineRef and taskSpec are mutually exclusive.
	if (ehs.PipelineRef != nil && ehs.PipelineRef.Name != "") && ehs.PipelineSpec != nil {
		return apis.ErrMultipleOneOf("spec.pipelineRef", "spec.pipelineSpec")
	}
	// Check that one of pipelineRef and taskSpec is present.
	if (ehs.PipelineRef == nil || ehs.PipelineRef.Name == "") && ehs.PipelineSpec == nil {
		return apis.ErrMissingOneOf("spec.pipelineRef", "spec.pipelineSpec")
	}
	// Validate PipelineSpec if it's present
	if ehs.PipelineSpec != nil {
		if err := ehs.PipelineSpec.Validate(ctx); err != nil {
			return err.ViaField("spec.pipelineSpec")
		}
	}
	if ehs.PipelineRef != nil && ehs.PipelineRef.Name != "" {
		// pipelineRef name must be a valid k8s name
		if errSlice := validation.IsQualifiedName(ehs.PipelineRef.Name); len(errSlice) != 0 {
			return apis.ErrInvalidValue(strings.Join(errSlice, ","), "spec.pipelineRef.name")
		}
	}
	return nil
}
