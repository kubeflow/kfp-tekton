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

package pipelinelooprun

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	fakeclient "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/client/fake"
	fakepipelineloopinformer "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/informers/pipelineloop/v1alpha1/pipelineloop/fake"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

var (
	namespace = ""
	trueB     = true
)

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
}

func loopRunning(run *v1alpha1.Run) *v1alpha1.Run {
	runWithStatus := run.DeepCopy()
	runWithStatus.Status.InitializeConditions()
	runWithStatus.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(), "")
	return runWithStatus
}

func requestCancel(run *v1alpha1.Run) *v1alpha1.Run {
	runWithCancelStatus := run.DeepCopy()
	runWithCancelStatus.Spec.Status = v1alpha1.RunSpecStatusCancelled
	return runWithCancelStatus
}

func running(tr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	trWithStatus := tr.DeepCopy()
	trWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionUnknown,
		Reason: v1beta1.PipelineRunReasonRunning.String(),
	})
	return trWithStatus
}

func successful(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
		Message: "All Steps have completed executing",
	})
	return prWithStatus
}

func successfulWithSkipedTasks(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
		Message: "Tasks Completed: 2 (Failed: 0, Cancelled 0), Skipped: 1",
	})
	prWithStatus.Status.SkippedTasks = []v1beta1.SkippedTask{{
		Name: "task-fail",
	}}
	return prWithStatus
}

func failed(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  v1beta1.PipelineRunReasonFailed.String(),
		Message: "Something went wrong",
	})
	return prWithStatus
}

// getPipelineLoopController returns an instance of the PipelineLoop controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func getPipelineLoopController(t *testing.T, d test.Data, pipelineloops []*pipelineloopv1alpha1.PipelineLoop) (test.Assets, func()) {
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d)

	client := fakeclient.Get(ctx)
	client.PrependReactor("*", "pipelineloops", test.AddToInformer(t, fakepipelineloopinformer.Get(ctx).Informer().GetIndexer()))
	for _, tl := range pipelineloops {
		tl := tl.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := client.CustomV1alpha1().PipelineLoops(tl.Namespace).Create(ctx, tl, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := informer.NewInformedWatcher(c.Kube, system.Namespace())
	ctl := NewController(namespace)(ctx, configMapWatcher)

	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {})
	}
	if err := configMapWatcher.Start(ctx.Done()); err != nil {
		t.Fatalf("error starting configmap watcher: %v", err)
	}

	return test.Assets{
		Logger:     logging.FromContext(ctx),
		Controller: ctl,
		Clients:    c,
		Informers:  informers,
		Recorder:   controller.GetEventRecorder(ctx).(*record.FakeRecorder),
	}, cancel
}

func getCreatedPipelinerun(t *testing.T, clients test.Clients) *v1beta1.PipelineRun {
	t.Log("actions", clients.Pipeline.Actions())
	for _, a := range clients.Pipeline.Actions() {
		if a.GetVerb() == "create" {
			obj := a.(ktesting.CreateAction).GetObject()
			if pr, ok := obj.(*v1beta1.PipelineRun); ok {
				return pr
			}
		}
	}
	return nil
}

func checkEvents(fr *record.FakeRecorder, testName string, wantEvents []string) error {
	// The fake recorder runs in a go routine, so the timeout is here to avoid waiting
	// on the channel forever if fewer than expected events are received.
	// We only hit the timeout in case of failure of the test, so the actual value
	// of the timeout is not so relevant. It's only used when tests are going to fail.
	timer := time.NewTimer(1 * time.Second)
	foundEvents := []string{}
	for ii := 0; ii < len(wantEvents)+1; ii++ {
		// We loop over all the events that we expect. Once they are all received
		// we exit the loop. If we never receive enough events, the timeout takes us
		// out of the loop.
		select {
		case event := <-fr.Events:
			foundEvents = append(foundEvents, event)
			if ii > len(wantEvents)-1 {
				return fmt.Errorf(`Received extra event "%s" for test "%s"`, event, testName)
			}
			wantEvent := wantEvents[ii]
			if !(strings.HasPrefix(event, wantEvent)) {
				return fmt.Errorf(`Expected event "%s" but got "%s" instead for test "%s"`, wantEvent, event, testName)
			}
		case <-timer.C:
			if len(foundEvents) > len(wantEvents) {
				return fmt.Errorf(`Received %d events but %d expected for test "%s". Found events: %#v`, len(foundEvents), len(wantEvents), testName, foundEvents)
			}
		}
	}
	return nil
}

func checkRunCondition(t *testing.T, run *v1alpha1.Run, expectedStatus corev1.ConditionStatus, expectedReason pipelineloopv1alpha1.PipelineLoopRunReason) {
	condition := run.Status.GetCondition(apis.ConditionSucceeded)
	if condition == nil {
		t.Error("Condition missing in Run")
	} else {
		if condition.Status != expectedStatus {
			t.Errorf("Expected Run status to be %v but was %v", expectedStatus, condition)
		}
		if condition.Reason != expectedReason.String() {
			t.Errorf("Expected reason to be %q but was %q", expectedReason.String(), condition.Reason)
		}
	}
	if run.Status.StartTime == nil {
		t.Errorf("Expected Run start time to be set but it wasn't")
	}
	if expectedStatus == corev1.ConditionUnknown {
		if run.Status.CompletionTime != nil {
			t.Errorf("Expected Run completion time to not be set but it was")
		}
	} else if run.Status.CompletionTime == nil {
		t.Errorf("Expected Run completion time to be set but it wasn't")
	}
}

func checkRunStatus(t *testing.T, run *v1alpha1.Run, expectedStatus map[string]pipelineloopv1alpha1.PipelineLoopPipelineRunStatus) {
	status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		t.Errorf("DecodeExtraFields error: %v", err.Error())
	}
	t.Log("pipelineruns", status.PipelineRuns)
	if len(status.PipelineRuns) != len(expectedStatus) {
		t.Errorf("Expected Run status to include %d PipelineRuns but found %d: %v", len(expectedStatus), len(status.PipelineRuns), status.PipelineRuns)
		return
	}
	for expectedPipelineRunName, expectedPipelineRunStatus := range expectedStatus {
		actualPipelineRunStatus, exists := status.PipelineRuns[expectedPipelineRunName]
		if !exists {
			t.Errorf("Expected Run status to include PipelineRun status for PipelineRun %s", expectedPipelineRunName)
			continue
		}
		if actualPipelineRunStatus.Iteration != expectedPipelineRunStatus.Iteration {
			t.Errorf("Run status for PipelineRun %s has iteration number %d instead of %d",
				expectedPipelineRunName, actualPipelineRunStatus.Iteration, expectedPipelineRunStatus.Iteration)
		}
		if d := cmp.Diff(expectedPipelineRunStatus.Status, actualPipelineRunStatus.Status, cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)); d != "" {
			t.Errorf("Run status for PipelineRun %s is incorrect. Diff %s", expectedPipelineRunName, diff.PrintWantGot(d))
		}
	}
}

var aPipeline = &v1beta1.Pipeline{
	ObjectMeta: metav1.ObjectMeta{Name: "a-pipeline", Namespace: "foo"},
	Spec: v1beta1.PipelineSpec{
		Params: []v1beta1.ParamSpec{{
			Name: "current-item",
			Type: v1beta1.ParamTypeString,
		}, {
			Name: "additional-parameter",
			Type: v1beta1.ParamTypeString,
		}},
		Tasks: []v1beta1.PipelineTask{{
			Name: "mytask",
			TaskSpec: &v1beta1.EmbeddedTask{
				TaskSpec: v1beta1.TaskSpec{
					Steps: []v1beta1.Step{{
						Container: corev1.Container{Name: "foo", Image: "bar"},
					}},
				},
			},
		}},
	},
}

var aPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:  &v1beta1.PipelineRef{Name: "a-pipeline"},
		IterateParam: "current-item",
	},
}

var nPipeline = &v1beta1.Pipeline{
	ObjectMeta: metav1.ObjectMeta{Name: "n-pipeline", Namespace: "foo"},
	Spec: v1beta1.PipelineSpec{
		Params: []v1beta1.ParamSpec{{
			Name: "iteration",
			Type: v1beta1.ParamTypeString,
		}, {
			Name: "additional-parameter",
			Type: v1beta1.ParamTypeString,
		}},
		Tasks: []v1beta1.PipelineTask{{
			Name: "mytask",
			TaskSpec: &v1beta1.EmbeddedTask{
				TaskSpec: v1beta1.TaskSpec{
					Steps: []v1beta1.Step{{
						Container: corev1.Container{Name: "foo", Image: "bar"},
					}},
				},
			},
		}},
	},
}

var nPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "n-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:    &v1beta1.PipelineRef{Name: "n-pipeline"},
		IterateNumeric: "iteration",
	},
}

var aPipelineLoopWithInlineTask = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-pipelineloop-with-inline-task", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineSpec: &v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "mytask",
				TaskSpec: &v1beta1.EmbeddedTask{
					TaskSpec: v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "current-item",
							Type: v1beta1.ParamTypeString,
						}, {
							Name: "additional-parameter",
							Type: v1beta1.ParamTypeString,
						}},
						Steps: []v1beta1.Step{{
							Container: corev1.Container{Name: "foo", Image: "bar"},
						}},
					},
				},
			}},
		},
		IterateParam: "current-item",
		Timeout:      &metav1.Duration{Duration: 5 * time.Minute},
	},
}

var runPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "a-pipelineloop",
			"tekton.dev/pipeline":            "pr-loop-example",
			"tekton.dev/pipelineRun":         "pr-loop-example",
			"tekton.dev/pipelineTask":        "loop-task",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var conditionRunPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":    "myTestLabelValue",
			"last-loop-task": "task-fail",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithInDictParams = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "a-pipelineloop",
			"tekton.dev/pipeline":            "pr-loop-example",
			"tekton.dev/pipelineRun":         "pr-loop-example",
			"tekton.dev/pipelineTask":        "loop-task",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `[{"a":1,"b":2}, {"a":2,"b":1}]`},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithInStringParams = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "a-pipelineloop",
			"tekton.dev/pipeline":            "pr-loop-example",
			"tekton.dev/pipelineRun":         "pr-loop-example",
			"tekton.dev/pipelineTask":        "loop-task",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `["item1", "item2"]`},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithIterateNumeric = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "n-pipelineloop",
			"tekton.dev/pipeline":            "pr-loop-example",
			"tekton.dev/pipelineRun":         "pr-loop-example",
			"tekton.dev/pipelineTask":        "loop-task",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "from",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `1`},
		}, {
			Name:  "step",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `1`},
		}, {
			Name:  "to",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `3`},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "n-pipelineloop",
		},
	},
}

var runPipelineLoopWithInlineTask = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-with-inline-task",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel": "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop-with-inline-task",
		},
	},
}

var runWithMissingPipelineLoopName = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-pipelineloop-missing",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			// missing Name
		},
	},
}

var runWithNonexistentPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-pipelineloop-not-found",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "no-such-pipelineloop",
		},
	},
}

var runWithMissingIterateParam = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-missing-iterate-param",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		// current-item, which is the iterate parameter, is missing from parameters
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runWithIterateParamNotAnArray = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-iterate-param-not-an-array",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			// Value of iteration parameter must be an array so this is an error.
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var expectedPipelineRunIterationDict = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "current-item-subvar-a",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "1"},
		}, {
			Name:  "current-item-subvar-b",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "2"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

var expectedPipelineRunIteration1 = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

var expectedConditionPipelineRunIteration1 = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
			"last-loop-task":                          "task-fail",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

var expectedPipelineRunIterateNumeric1 = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "n-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
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

// Note: The pipelinerun for the second iteration has the same random suffix as the first due to the resetting of the seed on each test.
var expectedPipelineRunIteration2 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00002-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "2",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

var expectedPipelineRunWithInlineTaskIteration1 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-with-inline-task-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop-with-inline-task",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop-with-inline-task",
			"tekton.dev/run":                          "run-pipelineloop-with-inline-task",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineSpec: &v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "mytask",
				TaskSpec: &v1beta1.EmbeddedTask{
					TaskSpec: v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "current-item",
							Type: v1beta1.ParamTypeString,
						}, {
							Name: "additional-parameter",
							Type: v1beta1.ParamTypeString,
						}},
						Steps: []v1beta1.Step{{
							Container: corev1.Container{Name: "foo", Image: "bar"},
						}},
					},
				},
			}},
		},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Timeout: &metav1.Duration{Duration: 5 * time.Minute},
	},
}

func TestReconcilePipelineLoopRun(t *testing.T) {

	testcases := []struct {
		name                 string
		pipeline             *v1beta1.Pipeline
		pipelineloop         *pipelineloopv1alpha1.PipelineLoop
		run                  *v1alpha1.Run
		pipelineruns         []*v1beta1.PipelineRun
		expectedStatus       corev1.ConditionStatus
		expectedReason       pipelineloopv1alpha1.PipelineLoopRunReason
		expectedPipelineruns []*v1beta1.PipelineRun
		expectedEvents       []string
	}{{
		name:                 "Reconcile a new run with a pipelineloop that references a pipeline",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoop,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and a dict params",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithInDictParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterationDict},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and a string params",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithInStringParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a run with condition pipelinerun, and the first PipelineRun condition check failed",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  loopRunning(conditionRunPipelineLoop),
		pipelineruns:         []*v1beta1.PipelineRun{successfulWithSkipedTasks(expectedConditionPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionTrue,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded,
		expectedPipelineruns: []*v1beta1.PipelineRun{successfulWithSkipedTasks(expectedConditionPipelineRunIteration1)},
		expectedEvents:       []string{"Normal Succeeded PipelineRuns completed successfully with the conditions are met"},
	}, {
		name:                 "Reconcile a new run with iterateNumeric defined",
		pipeline:             nPipeline,
		pipelineloop:         nPipelineLoop,
		run:                  runPipelineLoopWithIterateNumeric,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterateNumeric1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop that contains an inline task",
		pipelineloop:         aPipelineLoopWithInlineTask,
		run:                  runPipelineLoopWithInlineTask,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithInlineTaskIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a run after the first PipelineRun has succeeded.",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  loopRunning(runPipelineLoop),
		pipelineruns:         []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1), expectedPipelineRunIteration2},
		expectedEvents:       []string{"Normal Running Iterations completed: 1"},
	}, {
		name:                 "Reconcile a run after all PipelineRuns have succeeded",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  loopRunning(runPipelineLoop),
		pipelineruns:         []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1), successful(expectedPipelineRunIteration2)},
		expectedStatus:       corev1.ConditionTrue,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded,
		expectedPipelineruns: []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1), successful(expectedPipelineRunIteration2)},
		expectedEvents:       []string{"Normal Succeeded All PipelineRuns completed successfully"},
	}, {
		name:                 "Reconcile a run after the first PipelineRun has failed",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  loopRunning(runPipelineLoop),
		pipelineruns:         []*v1beta1.PipelineRun{failed(expectedPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailed,
		expectedPipelineruns: []*v1beta1.PipelineRun{failed(expectedPipelineRunIteration1)},
		expectedEvents:       []string{"Warning Failed PipelineRun " + expectedPipelineRunIteration1.Name + " has failed"},
	}, {
		name:                 "Reconcile a cancelled run while the first PipelineRun is running",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  requestCancel(loopRunning(runPipelineLoop)),
		pipelineruns:         []*v1beta1.PipelineRun{running(expectedPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{running(expectedPipelineRunIteration1)},
		expectedEvents:       []string{"Normal Running Cancelling PipelineRun " + expectedPipelineRunIteration1.Name},
	}, {
		name:                 "Reconcile a cancelled run after the first PipelineRun has failed",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  requestCancel(loopRunning(runPipelineLoop)),
		pipelineruns:         []*v1beta1.PipelineRun{failed(expectedPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonCancelled,
		expectedPipelineruns: []*v1beta1.PipelineRun{failed(expectedPipelineRunIteration1)},
		expectedEvents:       []string{"Warning Failed Run " + runPipelineLoop.Namespace + "/" + runPipelineLoop.Name + " was cancelled"},
	}, {
		name:                 "Reconcile a cancelled run after the first PipelineRun has succeeded",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  requestCancel(loopRunning(runPipelineLoop)),
		pipelineruns:         []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1)},
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonCancelled,
		expectedPipelineruns: []*v1beta1.PipelineRun{successful(expectedPipelineRunIteration1)},
		expectedEvents:       []string{"Warning Failed Run " + runPipelineLoop.Namespace + "/" + runPipelineLoop.Name + " was cancelled"},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			optionalPipeline := []*v1beta1.Pipeline{tc.pipeline}
			if tc.pipeline == nil {
				optionalPipeline = nil
			}

			d := test.Data{
				Runs:         []*v1alpha1.Run{tc.run},
				Pipelines:    optionalPipeline,
				PipelineRuns: tc.pipelineruns,
			}

			testAssets, _ := getPipelineLoopController(t, d, []*pipelineloopv1alpha1.PipelineLoop{tc.pipelineloop})
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run has the expected status and reason.
			checkRunCondition(t, reconciledRun, tc.expectedStatus, tc.expectedReason)

			// Verify that a PipelineRun was or was not created depending on the test.
			// If the number of expected PipelineRuns is greater than the original number of PipelineRuns
			// then the test expects a new PipelineRun to be created.  The new PipelineRun must be the
			// last one in the list of expected PipelineRuns.
			createdPipelinerun := getCreatedPipelinerun(t, clients)
			if len(tc.expectedPipelineruns) > len(tc.pipelineruns) {
				if createdPipelinerun == nil {
					t.Errorf("A PipelineRun should have been created but was not")
				} else {
					if d := cmp.Diff(tc.expectedPipelineruns[len(tc.expectedPipelineruns)-1], createdPipelinerun); d != "" {
						t.Errorf("Expected PipelineRun was not created. Diff %s", diff.PrintWantGot(d))
					}
				}
			} else {
				if createdPipelinerun != nil {
					t.Errorf("A PipelineRun was created which was not expected")
				}
			}

			// Verify Run status contains status for all PipelineRuns.
			expectedPipelineRuns := map[string]pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{}
			for i, pr := range tc.expectedPipelineruns {
				expectedPipelineRuns[pr.Name] = pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{Iteration: i + 1, Status: &pr.Status}
			}
			checkRunStatus(t, reconciledRun, expectedPipelineRuns)

			// Verify expected events were created.
			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}

func TestReconcilePipelineLoopRunFailures(t *testing.T) {
	testcases := []struct {
		name         string
		pipelineloop *pipelineloopv1alpha1.PipelineLoop
		run          *v1alpha1.Run
		reason       pipelineloopv1alpha1.PipelineLoopRunReason
		wantEvents   []string
	}{{
		name:   "missing PipelineLoop name",
		run:    runWithMissingPipelineLoopName,
		reason: pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop,
		wantEvents: []string{
			"Normal Started ",
			"Warning Failed Missing spec.ref.name for Run",
		},
	}, {
		name:   "nonexistent PipelineLoop",
		run:    runWithNonexistentPipelineLoop,
		reason: pipelineloopv1alpha1.PipelineLoopRunReasonCouldntGetPipelineLoop,
		wantEvents: []string{
			"Normal Started ",
			"Warning Failed Error retrieving PipelineLoop",
		},
	}, {
		name:         "missing iterate parameter",
		pipelineloop: aPipelineLoop,
		run:          runWithMissingIterateParam,
		reason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: The iterate parameter "current-item" was not found`,
		},
	}, {
		name:         "iterate parameter not an array",
		pipelineloop: aPipelineLoop,
		run:          runWithIterateParamNotAnArray,
		reason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: The value of the iterate parameter "current-item" can not transfer to array`,
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			d := test.Data{
				Runs: []*v1alpha1.Run{tc.run},
			}

			optionalPipelineLoop := []*pipelineloopv1alpha1.PipelineLoop{tc.pipelineloop}
			if tc.pipelineloop == nil {
				optionalPipelineLoop = nil
			}

			testAssets, _ := getPipelineLoopController(t, d, optionalPipelineLoop)
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run is in Failed status and both the start time and the completion time are set.
			checkRunCondition(t, reconciledRun, corev1.ConditionFalse, tc.reason)
			if reconciledRun.Status.StartTime == nil {
				t.Fatalf("Expected Run start time to be set but it wasn't")
			}
			if reconciledRun.Status.CompletionTime == nil {
				t.Fatalf("Expected Run completion time to be set but it wasn't")
			}

			if err := checkEvents(testAssets.Recorder, tc.name, tc.wantEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
