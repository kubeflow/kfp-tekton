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

var _ apis.Validatable = (*KfpTask)(nil)

// Validate KfpTask
func (kt *KfpTask) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(kt.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return kt.Spec.Validate(ctx)
}

// Validate KfpTaskSpec
func (kts *KfpTaskSpec) Validate(ctx context.Context) *apis.FieldError {
	// Validate Task reference or inline task spec.
	if err := validateTaskSpec(ctx, kts); err != nil {
		return err
	}
	return nil
}

func validateTaskSpec(ctx context.Context, kts *KfpTaskSpec) *apis.FieldError {
	// taskRef and taskSpec are mutually exclusive.
	if (kts.TaskRef != nil && kts.TaskRef.Name != "") && kts.TaskSpec != nil {
		return apis.ErrMultipleOneOf("spec.taskRef", "spec.taskSpec")
	}
	// Check that one of taskRef and taskSpec is present.
	if (kts.TaskRef == nil || kts.TaskRef.Name == "") && kts.TaskSpec == nil {
		return apis.ErrMissingOneOf("spec.taskRef", "spec.taskSpec")
	}
	// Validate PipelineSpec if it's present
	if kts.TaskSpec != nil {
		if err := kts.TaskSpec.Validate(ctx); err != nil {
			return err.ViaField("spec.taskSpec")
		}
	}
	if kts.TaskRef != nil && kts.TaskRef.Name != "" {
		// TaskRef name must be a valid k8s name
		if errSlice := validation.IsQualifiedName(kts.TaskRef.Name); len(errSlice) != 0 {
			return apis.ErrInvalidValue(strings.Join(errSlice, ","), "spec.taskRef.name")
		}
	}
	return nil
}
