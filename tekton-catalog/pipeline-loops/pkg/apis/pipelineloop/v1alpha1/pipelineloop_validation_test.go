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

package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/test/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestPipelineLoop_Validate_Success(t *testing.T) {
	tests := []struct {
		name string
		tl   *pipelineloopv1alpha1.PipelineLoop
	}{{
		name: "pipelineRef",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				PipelineRef: &tektonv1.PipelineRef{Name: "mypipeline"},
			},
		},
	}, {
		name: "pipelineSpecWithoutParam",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				IterateParam: "messages",
				PipelineSpec: &tektonv1.PipelineSpec{
					Tasks: []tektonv1.PipelineTask{{
						Name: "mytask",
						TaskSpec: &tektonv1.EmbeddedTask{
							TaskSpec: tektonv1.TaskSpec{
								Steps: []tektonv1.Step{{
									Name: "foo", Image: "bar",
								}},
							},
						},
					}},
				},
			},
		},
	}, {
		name: "pipelineSpecWithParams",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				IterateParam: "messages",
				PipelineSpec: &tektonv1.PipelineSpec{
					Params: []tektonv1.ParamSpec{{
						Name: "messages",
						Type: tektonv1.ParamTypeString,
					}, {
						Name: "additional-parameter",
						Type: tektonv1.ParamTypeString,
					}},
					Tasks: []tektonv1.PipelineTask{{
						Name: "mytask",
						Params: []tektonv1.Param{{
							Name:  "messages",
							Value: tektonv1.ParamValue{},
						}, {
							Name:  "additional-parameter",
							Value: tektonv1.ParamValue{},
						}},
						TaskSpec: &tektonv1.EmbeddedTask{
							TaskSpec: tektonv1.TaskSpec{
								Params: []tektonv1.ParamSpec{{
									Name: "messages",
									Type: tektonv1.ParamTypeString,
								}, {
									Name: "additional-parameter",
									Type: tektonv1.ParamTypeString,
								}},
								Steps: []tektonv1.Step{{
									Name: "foo", Image: "bar",
								}},
							},
						},
					}},
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.tl.SetDefaults(ctx)
			err := tc.tl.Validate(ctx)
			if err != nil {
				t.Errorf("Unexpected error for %s: %s", tc.name, err)
			}
		})
	}
}

func TestPipelineLoop_Validate_Error(t *testing.T) {
	tests := []struct {
		name          string
		tl            *pipelineloopv1alpha1.PipelineLoop
		expectedError apis.FieldError
	}{{
		name: "no pipelineRef or pipelineSpec",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec:       pipelineloopv1alpha1.PipelineLoopSpec{},
		},
		expectedError: apis.FieldError{
			Message: "expected exactly one, got neither",
			Paths:   []string{"spec.pipelineRef", "spec.pipelineSpec"},
		},
	}, {
		name: "both pipelineRef and pipelineSpec",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				PipelineRef: &tektonv1.PipelineRef{Name: "mypipeline"},
				PipelineSpec: &tektonv1.PipelineSpec{
					Tasks: []tektonv1.PipelineTask{{
						Name: "mytask",
						TaskSpec: &tektonv1.EmbeddedTask{
							TaskSpec: tektonv1.TaskSpec{
								Steps: []tektonv1.Step{{
									Name: "foo", Image: "bar",
								}},
							},
						},
					}},
				},
			},
		},
		expectedError: apis.FieldError{
			Message: "expected exactly one, got both",
			Paths:   []string{"spec.pipelineRef", "spec.pipelineSpec"},
		},
	}, {
		name: "invalid pipelineRef",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				PipelineRef: &tektonv1.PipelineRef{Name: "_bad"},
			},
		},
		expectedError: apis.FieldError{
			Message: "invalid value: name part must consist of alphanumeric characters, '-', '_' or '.', and must start " +
				"and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for " +
				"validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
			Paths: []string{"spec.pipelineRef.name"},
		},
	}, {
		name: "invalid pipelineSpec",
		tl: &pipelineloopv1alpha1.PipelineLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "pipelineloop"},
			Spec: pipelineloopv1alpha1.PipelineLoopSpec{
				PipelineSpec: &tektonv1.PipelineSpec{
					Tasks: []tektonv1.PipelineTask{{
						Name: "mytask",
						TaskSpec: &tektonv1.EmbeddedTask{
							TaskSpec: tektonv1.TaskSpec{
								Steps: []tektonv1.Step{{
									Name: "bad@name!", Image: "bar",
								}},
							},
						},
					}},
				},
			},
		},
		expectedError: apis.FieldError{
			Message: `invalid value "bad@name!"`,
			Details: "Task step name must be a valid DNS Label, For more info refer to https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			Paths:   []string{"spec.pipelineSpec.tasks[0].taskSpec.steps[0].name"},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.tl.SetDefaults(ctx)
			err := tc.tl.Validate(ctx)
			if err == nil {
				t.Errorf("Expected an Error but did not get one for %s", tc.name)
			} else {
				if d := cmp.Diff(tc.expectedError.Error(), err.Error(), cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
					t.Errorf("Error is different from expected for %s. diff %s", tc.name, diff.PrintWantGot(d))
				}
			}
		})
	}
}
