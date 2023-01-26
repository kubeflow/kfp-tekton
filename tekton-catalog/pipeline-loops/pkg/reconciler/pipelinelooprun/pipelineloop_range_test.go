/*
Copyright 2022 The Kubeflow Authors

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

package pipelinelooprun

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "knative.dev/pkg/system/testing"
)

var expectedPipelineRunWithRange1 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "n-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": "-1",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "n-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "iteration",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "-1"},
		}},
	},
}

var expectedPipelineRunWithRange2 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "n-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": "1",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "n-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "iteration",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "1"},
		}},
	},
}
var expectedPipelineRunWithRange3 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "n-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": "0",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "n-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "iteration",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "0"},
		}},
	},
}

func TestReconcilePipelineLoopRunRange(t *testing.T) {

	testcases := []struct {
		name                 string
		from                 string
		to                   string
		step                 string
		expectedStatus       corev1.ConditionStatus
		expectedReason       pipelineloopv1alpha1.PipelineLoopRunReason
		expectedPipelineruns []*v1beta1.PipelineRun
		expectedEvents       []string
	}{{
		name:                 "Case from = to",
		from:                 "1",
		to:                   "1",
		step:                 "1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange2},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from 0 to 0 and non zero step increment",
		from:                 "0",
		step:                 "1",
		to:                   "0",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange3},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from < to and +ve step increment",
		from:                 "-1",
		step:                 "1",
		to:                   "0",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange1},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from < to and step == 0",
		from:                 "-1",
		step:                 "0",
		to:                   "0",
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		expectedPipelineruns: []*v1beta1.PipelineRun{},
		expectedEvents:       []string{"Normal Started ", "Warning Failed Cannot determine number of iterations: invalid values step: 0 found in runs"},
	}, {
		name:                 "Case to - from < step and step > 0",
		from:                 "1",
		step:                 "1",
		to:                   "-1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange2},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case to - from < step and step > 0",
		from:                 "0",
		step:                 "1",
		to:                   "-1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange3},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from = to",
		from:                 "-1",
		step:                 "1",
		to:                   "-1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange1},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from > to and non -ve step increment",
		from:                 "1",
		step:                 "0",
		to:                   "-1",
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		expectedPipelineruns: []*v1beta1.PipelineRun{},
		expectedEvents:       []string{"Normal Started ", "Warning Failed Cannot determine number of iterations: invalid values step: 0 found in runs"},
	}, {
		name:                 "Case step == 0",
		from:                 "0",
		step:                 "0",
		to:                   "-1",
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		expectedPipelineruns: []*v1beta1.PipelineRun{},
		expectedEvents:       []string{"Normal Started ", "Warning Failed Cannot determine number of iterations: invalid values step: 0 found in runs"},
	}, {
		name:                 "Case from > to and -ve step increment",
		from:                 "1",
		step:                 "-1",
		to:                   "-1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange2},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from > to and -ve step increment",
		from:                 "0",
		step:                 "-1",
		to:                   "-1",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange3},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from == 0\n",
		from:                 "0\n",
		step:                 "-1",
		to:                   "-1\n",
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithRange3},
		expectedEvents:       []string{"Normal Started ", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Case from == abc\n",
		from:                 "abc",
		step:                 "-1",
		to:                   "edf",
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		expectedPipelineruns: []*v1beta1.PipelineRun{},
		expectedEvents:       []string{"Normal Started ", "Warning Failed Cannot determine number of iterations: input \"to\" is not a number"},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()
			run := specifyLoopRange(tc.from, tc.to, tc.step, runPipelineLoopWithIterateNumeric)
			d := test.Data{
				Runs:         []*v1alpha1.Run{run},
				Pipelines:    []*v1beta1.Pipeline{nPipeline},
				PipelineRuns: []*v1beta1.PipelineRun{},
			}

			testAssets, _ := getPipelineLoopController(t, d, []*pipelineloopv1alpha1.PipelineLoop{nPipelineLoop})
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(ctx, getRunName(run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(run.Namespace).Get(ctx, run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run has the expected status and reason.
			checkRunCondition(t, reconciledRun, tc.expectedStatus, tc.expectedReason)

			// Verify that a PipelineRun was or was not created depending on the test.
			// If the number of expected PipelineRuns is greater than the original number of PipelineRuns
			// then the test expects a new PipelineRun to be created.  The new PipelineRun must be the
			// last one in the list of expected PipelineRuns.
			createdPipelineruns := getCreatedPipelinerun(t, clients)
			// All the arrays and sub arrays are sorted to ensure there are no sporadic failures
			// resulting from mismatch due to different ordering of items.
			sort.Slice(createdPipelineruns, func(i, j int) bool {
				return createdPipelineruns[i].Name < createdPipelineruns[j].Name
			})
			for _, createdPipelinerun := range createdPipelineruns {
				sort.Slice(createdPipelinerun.Spec.Params, func(i, j int) bool {
					return createdPipelinerun.Spec.Params[i].Name < createdPipelinerun.Spec.Params[j].Name
				})
				if createdPipelinerun.Spec.PipelineSpec != nil {
					sort.Slice(createdPipelinerun.Spec.PipelineSpec.Params, func(i, j int) bool {
						return createdPipelinerun.Spec.PipelineSpec.Params[i].Name < createdPipelinerun.Spec.PipelineSpec.Params[j].Name
					})
					sort.Slice(createdPipelinerun.Spec.PipelineSpec.Tasks, func(i, j int) bool {
						return createdPipelinerun.Spec.PipelineSpec.Tasks[i].Name < createdPipelinerun.Spec.PipelineSpec.Tasks[j].Name
					})
					for _, t := range createdPipelinerun.Spec.PipelineSpec.Tasks {
						sort.Slice(t.Params, func(i, j int) bool {
							return t.Params[i].Name < t.Params[j].Name
						})
						if t.TaskSpec != nil {
							sort.Slice(t.TaskSpec.Params, func(i, j int) bool {
								return t.TaskSpec.Params[i].Name < t.TaskSpec.Params[j].Name
							})
						}
					}
				}
			}
			if len(tc.expectedPipelineruns) > 0 {
				if len(createdPipelineruns) == 0 {
					t.Errorf("A PipelineRun should have been created but was not")
				} else {
					pipelineRunsExpectedToBeCreated := make([]*v1beta1.PipelineRun, len(createdPipelineruns))
					i := 0
					for _, pr := range tc.expectedPipelineruns {
						if pr.Labels["deleted"] != "True" {
							pipelineRunsExpectedToBeCreated[i] = pr
							i = i + 1 // skip the pr that were retried.
						}
					}

					if d := cmp.Diff(pipelineRunsExpectedToBeCreated, createdPipelineruns); d != "" {
						t.Errorf("Expected PipelineRun was not created. Diff %s", diff.PrintWantGot(d))
					}
				}
			} else {
				if len(createdPipelineruns) > 0 {
					t.Errorf("A PipelineRun was created which was not expected")
				}
			}

			// Verify expected events were created.
			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
