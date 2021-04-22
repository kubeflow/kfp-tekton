/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*PipelineLoop)(nil)

// Validate PipelineLoop
func (tl *PipelineLoop) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tl.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return tl.Spec.Validate(ctx)
}

// Validate PipelineLoopSpec
func (tls *PipelineLoopSpec) Validate(ctx context.Context) *apis.FieldError {
	// Validate Task reference or inline task spec.
	if err := validateTask(ctx, tls); err != nil {
		return err
	}
	return nil
}

func validateTask(ctx context.Context, tls *PipelineLoopSpec) *apis.FieldError {
	// pipelineRef and taskSpec are mutually exclusive.
	if (tls.PipelineRef != nil && tls.PipelineRef.Name != "") && tls.PipelineSpec != nil {
		return apis.ErrMultipleOneOf("spec.pipelineRef", "spec.pipelineSpec")
	}
	// Check that one of pipelineRef and taskSpec is present.
	if (tls.PipelineRef == nil || tls.PipelineRef.Name == "") && tls.PipelineSpec == nil {
		return apis.ErrMissingOneOf("spec.pipelineRef", "spec.pipelineSpec")
	}
	// Validate PipelineSpec if it's present
	if tls.PipelineSpec != nil {
		if err := tls.PipelineSpec.Validate(ctx); err != nil {
			return err.ViaField("spec.pipelineSpec")
		}
	}
	if tls.PipelineRef != nil && tls.PipelineRef.Name != "" {
		// pipelineRef name must be a valid k8s name
		if errSlice := validation.IsQualifiedName(tls.PipelineRef.Name); len(errSlice) != 0 {
			return apis.ErrInvalidValue(strings.Join(errSlice, ","), "spec.pipelineRef.name")
		}
	}
	return nil
}
