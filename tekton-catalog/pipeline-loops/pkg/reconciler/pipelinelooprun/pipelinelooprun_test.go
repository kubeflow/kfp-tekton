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
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

var (
	namespace = ""
	trueB     = true
)

func initCacheParams() {
	tmp := os.TempDir()
	params.DbDriver = "sqlite"
	params.DbName = tmp + "/testing.db"
	params.Timeout = 2 * time.Second
}

func init() {
	initCacheParams()
}

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
}

func loopRunning(run *v1alpha1.Run) *v1alpha1.Run {
	runWithStatus := run.DeepCopy()
	runWithStatus.Status.InitializeConditions()
	runWithStatus.Status.MarkRunRunning(pipelineloopv1alpha1.PipelineLoopRunReasonRunning.String(), "")
	return runWithStatus
}

func loopSucceeded(run *v1alpha1.Run) *v1alpha1.Run {
	runWithStatus := run.DeepCopy()
	runWithStatus.Status.InitializeConditions()
	runWithStatus.Status.MarkRunSucceeded(pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded.String(), "")
	return runWithStatus
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

func setRetries(run *v1alpha1.Run, retries int) *v1alpha1.Run {
	run.Spec.Retries = retries
	return run
}

func setDeleted(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	pr.Labels["deleted"] = "True"
	return pr
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

func getCreatedPipelinerun(t *testing.T, clients test.Clients) []*v1beta1.PipelineRun {
	t.Log("actions", clients.Pipeline.Actions())
	var createdPr []*v1beta1.PipelineRun
	for _, a := range clients.Pipeline.Actions() {
		if a.GetVerb() == "create" {
			obj := a.(ktesting.CreateAction).GetObject()
			if pr, ok := obj.(*v1beta1.PipelineRun); ok {
				createdPr = append(createdPr, pr)
			}
		}
	}
	return createdPr
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
		acturalIterationItem, error := json.Marshal(actualPipelineRunStatus.IterationItem)
		expectedIterationItem, _ := json.Marshal(expectedPipelineRunStatus.IterationItem)
		if error != nil || string(acturalIterationItem) != string(expectedIterationItem) {
			t.Errorf("Run status for PipelineRun %s has iteration item %v instead of %v",
				expectedPipelineRunName, actualPipelineRunStatus.IterationItem, expectedPipelineRunStatus.IterationItem)
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
						Name: "foo", Image: "bar",
					}},
				},
			},
		}},
	},
}

var aPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:           &v1beta1.PipelineRef{Name: "a-pipeline"},
		IterateParam:          "current-item",
		IterateParamSeparator: "separator",
	},
}

var aPipelineLoop2 = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-pipelineloop2", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:           &v1beta1.PipelineRef{Name: "a-pipeline"},
		IterateParam:          "current-item",
		IterateParamSeparator: "separator",
		IterationNumberParam:  "additional-parameter",
	},
}

var wsPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "ws-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:  &v1beta1.PipelineRef{Name: "a-pipeline"},
		IterateParam: "current-item",
		Workspaces: []v1beta1.WorkspaceBinding{{
			Name: "test",
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
				Items:                []corev1.KeyToPath{},
			},
		}},
	},
}

var newPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "new-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:        &v1beta1.PipelineRef{Name: "a-pipeline"},
		IterateParam:       "current-item",
		ServiceAccountName: "default",
		PodTemplate: &pod.PodTemplate{
			HostAliases: []corev1.HostAlias{{
				IP:        "0.0.0.0",
				Hostnames: []string{"localhost"},
			}},
			HostNetwork: true,
		},
		TaskRunSpecs: []v1beta1.PipelineTaskRunSpec{{
			PipelineTaskName:       "test-task",
			TaskServiceAccountName: "test",
			TaskPodTemplate: &pod.PodTemplate{
				HostAliases: []corev1.HostAlias{{
					IP:        "0.0.0.0",
					Hostnames: []string{"localhost"},
				}},
				HostNetwork: true,
			},
		}},
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
						Name: "foo", Image: "bar",
					}},
				},
			},
		}},
	},
}

var paraPipeline = &v1beta1.Pipeline{
	ObjectMeta: metav1.ObjectMeta{Name: "para-pipeline", Namespace: "foo"},
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
						Name: "foo", Image: "bar",
					}},
				},
			},
		}},
	},
}

func getInnerLoopByte(pipelineLoopSpec pipelineloopv1alpha1.PipelineLoopSpec) []byte {
	innerLoop, err := json.Marshal(pipelineLoopSpec)
	if err != nil {
		fmt.Println(fmt.Errorf("error while marshalling pipelineLoop %s", err.Error()).Error())
		panic(err)
	}
	return innerLoop
}

var ePipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "e-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineSpec: &paraPipeline.Spec,
		IterateParam: "current-item",
	},
}

var nestedPipeline = &v1beta1.Pipeline{
	ObjectMeta: metav1.ObjectMeta{Name: "nestedPipeline", Namespace: "foo"},
	Spec: v1beta1.PipelineSpec{
		Params: []v1beta1.ParamSpec{{
			Name: "additional-parameter",
			Type: v1beta1.ParamTypeString,
		}, {
			Name: "iteration",
			Type: v1beta1.ParamTypeString,
		}},
		Tasks: []v1beta1.PipelineTask{{
			Name: "mytask",
			TaskSpec: &v1beta1.EmbeddedTask{
				TypeMeta: runtime.TypeMeta{
					APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
					Kind:       pipelineloop.PipelineLoopControllerName,
				},
				Spec: runtime.RawExtension{
					Raw: getInnerLoopByte(ePipelineLoop.Spec),
				},
			},
		}},
	},
}

func setPipelineNestedStackDepth(pipeline *v1beta1.Pipeline, depth int) *v1beta1.Pipeline {
	pl := pipeline.DeepCopy()
	pl.Spec.Tasks[0].TaskSpec.Metadata.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
	return pl
}

var paraPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "para-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:  &v1beta1.PipelineRef{Name: "para-pipeline"},
		IterateParam: "current-item",
		Parallelism:  2,
	},
}

var nPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "n-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineRef:    &v1beta1.PipelineRef{Name: "n-pipeline"},
		IterateNumeric: "iteration",
	},
}

var nestedPipelineLoop = &pipelineloopv1alpha1.PipelineLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "nested-pipelineloop", Namespace: "foo"},
	Spec: pipelineloopv1alpha1.PipelineLoopSpec{
		PipelineSpec: &nestedPipeline.Spec,
		IterateParam: "current-item",
	},
}

func setPipelineLoopNestedStackDepth(pl *pipelineloopv1alpha1.PipelineLoop, depth int) *pipelineloopv1alpha1.PipelineLoop {
	plCopy := pl.DeepCopy()
	plCopy.Spec.PipelineSpec = &setPipelineNestedStackDepth(nestedPipeline, depth).Spec
	return plCopy
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
							Name: "foo", Image: "bar",
						}},
					},
				},
			}},
		},
		IterateParam: "current-item",
		Timeout:      &metav1.Duration{Duration: 5 * time.Minute},
	},
}

var runWsPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-ws-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "ws-pipelineloop",
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
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "ws-pipelineloop",
		},
	},
}

var runNewPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-new-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "new-pipelineloop",
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
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "new-pipelineloop",
		},
	},
}

var runNewPipelineLoopWithPodTemplateAndSA = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-a-pipelineloop",
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
		}},
		ServiceAccountName: "pipeline-runner",
		PodTemplate: &pod.PodTemplate{
			HostAliases: []corev1.HostAlias{{
				IP:        "0.0.0.0",
				Hostnames: []string{"localhost"},
			}},
			HostNetwork: true,
		},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
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
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoop2 = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel":                    "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop": "a-pipelineloop2",
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
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop2",
		},
	},
}

var runNestedPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nested-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTestLabel": "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation12": "myTestAnnotationValue12",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Spec: &v1alpha1.EmbeddedRunSpec{
			TypeMeta: runtime.TypeMeta{
				APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
				Kind:       pipelineloop.PipelineLoopControllerName,
			},
			Spec: runtime.RawExtension{
				Raw: getInnerLoopByte(nestedPipelineLoop.Spec),
			},
		},
	},
}

func setRunNestedStackDepth(run *v1alpha1.Run, depth int) *v1alpha1.Run {
	r := run.DeepCopy()
	r.Spec.Spec.Metadata.Annotations = map[string]string{MaxNestedStackDepthKey: fmt.Sprint(depth)}
	return r
}

var paraRunPipelineLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
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
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "para-pipelineloop",
		},
	},
}

var runPipelineLoopWithInDictParams = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop",
		Namespace: "foo",
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun": "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":   "pr-loop-example",
			"myTestLabel":                           "myTestLabelValue",
			"custom.tekton.dev/pipelineLoop":        "a-pipelineloop",
			"tekton.dev/pipeline":                   "pr-loop-example",
			"tekton.dev/pipelineRun":                "pr-loop-example",
			"tekton.dev/pipelineTask":               "loop-task",
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
		Ref: &v1beta1.TaskRef{
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
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithInStringSeparatorParams = &v1alpha1.Run{
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1|item2"},
		}, {
			Name:  "separator",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "|"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithSpaceSeparatorParams = &v1alpha1.Run{
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1 item2"},
		}, {
			Name:  "separator",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: " "},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithSpaceParam = &v1alpha1.Run{
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: " "},
		}, {
			Name:  "separator",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: ","},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

var runPipelineLoopWithDefaultSeparatorParams = &v1alpha1.Run{
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1,item2"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
	},
}

func specifyLoopRange(from, to, step string, r *v1alpha1.Run) *v1alpha1.Run {
	t := r.DeepCopy()
	for n, i := range r.Spec.Params {
		if i.Name == "from" {
			t.Spec.Params[n].Value = v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: from}
		}
		if i.Name == "to" {
			t.Spec.Params[n].Value = v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: to}
		}
		if i.Name == "step" {
			t.Spec.Params[n].Value = v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: step}
		}
	}
	return t
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
		Ref: &v1beta1.TaskRef{
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
		Ref: &v1beta1.TaskRef{
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
		Ref: &v1beta1.TaskRef{
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
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "no-such-pipelineloop",
		},
	},
}

var runWithInvalidRange = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-invalid-range",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		// current-item, which is the iterate parameter, is missing from parameters
		Params: []v1beta1.Param{{
			Name:  "from",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `-11`},
		}, {
			Name:  "step",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `1`},
		}, {
			Name:  "to",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: `-13`},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
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
		Ref: &v1beta1.TaskRef{
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
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": "{\"a\":1,\"b\":2}",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item-subvar-a",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "1"},
		}, {
			Name:  "current-item-subvar-b",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "2"},
		}},
	},
}

var expectedParaPipelineRun = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/originalPipelineRun":   "",
			"custom.tekton.dev/parentPipelineRun":     "",
			"custom.tekton.dev/pipelineLoop":          "para-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "para-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
}

var expectedParaPipelineRun1 = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00002-mz4c7",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "",
			"custom.tekton.dev/parentPipelineRun":     "",
			"custom.tekton.dev/pipelineLoop":          "para-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "2",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item2"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "para-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
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
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
}

var expectedPipelineRunIterationEmptySpace = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `" item1 "`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: " item1 "},
		}},
	},
}

var expectedPipelineRunIterationWithWhiteSpace = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `" "`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: " "},
		}},
	},
}

var expectedPipelineRunFailed = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-pipelineloop-00001-failed",
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
}

var expectedPipelineRunRetry = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
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

var expectedPipelineRunIterateNumeric2 = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": "-10",
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "n-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "iteration",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "-10"},
		}},
	},
}

var expectedPipelineRunIterateNumericParam = &v1beta1.PipelineRun{
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
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop2",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "1"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
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
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "2",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item2"`,
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

var expectedPipelineRunWithWorkSpace = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-ws-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-ws-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "ws-pipelineloop",
			"tekton.dev/run":                          "run-ws-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "a-pipeline",
		},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
		Workspaces: []v1beta1.WorkspaceBinding{{
			Name: "test",
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "test"},
				Items:                []corev1.KeyToPath{},
			},
		}},
	},
}

var expectedPipelineRunWithPodTemplateAndSA = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-a-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-a-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-a-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "a-pipeline",
		},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
		ServiceAccountName: "pipeline-runner",
		PodTemplate: &pod.PodTemplate{
			HostAliases: []corev1.HostAlias{{
				IP:        "0.0.0.0",
				Hostnames: []string{"localhost"},
			}},
			HostNetwork: true,
		},
	},
}

var expectedPipelineRunWithPodTemplate = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-new-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-new-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "pr-loop-example",
			"custom.tekton.dev/parentPipelineRun":     "pr-loop-example",
			"custom.tekton.dev/pipelineLoop":          "new-pipelineloop",
			"tekton.dev/run":                          "run-new-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "a-pipeline",
		},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
		ServiceAccountName: "default",
		PodTemplate: &pod.PodTemplate{
			HostAliases: []corev1.HostAlias{{
				IP:        "0.0.0.0",
				Hostnames: []string{"localhost"},
			}},
			HostNetwork: true,
		},
		TaskRunSpecs: []v1beta1.PipelineTaskRunSpec{{
			PipelineTaskName:       "test-task",
			TaskServiceAccountName: "test",
			TaskPodTemplate: &pod.PodTemplate{
				HostAliases: []corev1.HostAlias{{
					IP:        "0.0.0.0",
					Hostnames: []string{"localhost"},
				}},
				HostNetwork: true,
			},
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
			"custom.tekton.dev/originalPipelineRun":   "",
			"custom.tekton.dev/parentPipelineRun":     "",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop-with-inline-task",
			"tekton.dev/run":                          "run-pipelineloop-with-inline-task",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineSpec: &v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "mytask",
				TaskSpec: &v1beta1.EmbeddedTask{
					TaskSpec: v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "additional-parameter",
							Type: v1beta1.ParamTypeString,
						}, {
							Name: "current-item",
							Type: v1beta1.ParamTypeString,
						}},
						Steps: []v1beta1.Step{{
							Name: "foo", Image: "bar",
						}},
					},
				},
			}},
		},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
		Timeout: &metav1.Duration{Duration: 5 * time.Minute},
	},
}
var expectedNestedPipelineRun = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nested-pipelineloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "nested-pipelineloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/originalPipelineRun":   "",
			"custom.tekton.dev/parentPipelineRun":     "",
			"custom.tekton.dev/pipelineLoop":          "nested-pipelineloop",
			"tekton.dev/run":                          "nested-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation12": "myTestAnnotationValue12",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineSpec: &setPipelineNestedStackDepth(nestedPipeline, 29).Spec,
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
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
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
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
			"custom.tekton.dev/originalPipelineRun":   "",
			"custom.tekton.dev/parentPipelineRun":     "",
			"custom.tekton.dev/pipelineLoop":          "a-pipelineloop",
			"tekton.dev/run":                          "run-pipelineloop",
			"custom.tekton.dev/pipelineLoopIteration": "1",
			"myTestLabel":                             "myTestLabelValue",
			"last-loop-task":                          "task-fail",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
			"custom.tekton.dev/pipelineLoopCurrentIterationItem": `"item1"`,
		},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{Name: "a-pipeline"},
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}, {
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
}

var runPipelineLoopWithInStringSeparatorEmptySpaceParams = &v1alpha1.Run{
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: " item1 | item2 "},
		}, {
			Name:  "separator",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "|"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1beta1.TaskRef{
			APIVersion: pipelineloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       pipelineloop.PipelineLoopControllerName,
			Name:       "a-pipelineloop",
		},
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
		name:                 "Reconcile a new run with a pipelineloop and a string params with separator",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithInStringSeparatorParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and an empty space string params with separator",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithInStringSeparatorEmptySpaceParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterationEmptySpace},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and a string params with whitespace separator",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithSpaceSeparatorParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
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
		name:                 "Reconcile a new run with -ve numeric range defined",
		pipeline:             nPipeline,
		pipelineloop:         nPipelineLoop,
		run:                  specifyLoopRange("-10", "-15", "-1", runPipelineLoopWithIterateNumeric),
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterateNumeric2},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with iterationNumberParam defined",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop2,
		run:                  runPipelineLoop2,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterateNumericParam},
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
		name:                 "Reconcile a new run with a pipelineloop that contains a workspace",
		pipeline:             aPipeline,
		pipelineloop:         wsPipelineLoop,
		run:                  runWsPipelineLoop,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithWorkSpace},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
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
		name:                 "Reconcile a run with retries after the first PipelineRun has failed",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  loopRunning(setRetries(runPipelineLoop, 1)),
		pipelineruns:         []*v1beta1.PipelineRun{failed(expectedPipelineRunFailed)},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{setDeleted(failed(expectedPipelineRunFailed)), expectedPipelineRunRetry},
	}, {
		name:                 "Reconcile a new run with a pipelineloop with Parallelism specified",
		pipeline:             paraPipeline,
		pipelineloop:         paraPipelineLoop,
		run:                  paraRunPipelineLoop,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedParaPipelineRun, expectedParaPipelineRun1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a nested pipelineloop",
		pipeline:             nestedPipeline,
		pipelineloop:         nestedPipelineLoop,
		run:                  runNestedPipelineLoop,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedNestedPipelineRun},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a recursive pipelineloop with max nested stack depth 0",
		pipeline:             setPipelineNestedStackDepth(nestedPipeline, 0),
		pipelineloop:         setPipelineLoopNestedStackDepth(nestedPipelineLoop, 0),
		run:                  setRunNestedStackDepth(runNestedPipelineLoop, 0),
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionFalse,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonStackLimitExceeded,
		expectedPipelineruns: []*v1beta1.PipelineRun{},
		expectedEvents:       []string{"Normal Started ", "Warning Failed nested stack depth limit reached."},
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
		name:                 "Reconcile a new run with a pipelineloop that contains a PodTemplate, ServiceAccountName, TaskRunSpecs",
		pipeline:             aPipeline,
		pipelineloop:         newPipelineLoop,
		run:                  runNewPipelineLoop,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithPodTemplate},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run that contains a PodTemplate, ServiceAccountName",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runNewPipelineLoopWithPodTemplateAndSA,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunWithPodTemplateAndSA},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and a string params without separator",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithDefaultSeparatorParams,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIteration1},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:                 "Reconcile a new run with a pipelineloop and a string params without separator",
		pipeline:             aPipeline,
		pipelineloop:         aPipelineLoop,
		run:                  runPipelineLoopWithSpaceParam,
		pipelineruns:         []*v1beta1.PipelineRun{},
		expectedStatus:       corev1.ConditionUnknown,
		expectedReason:       pipelineloopv1alpha1.PipelineLoopRunReasonRunning,
		expectedPipelineruns: []*v1beta1.PipelineRun{expectedPipelineRunIterationWithWhiteSpace},
		expectedEvents:       []string{"Normal Started", "Normal Running Iterations completed: 0"},
	},
	}

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
			if len(tc.expectedPipelineruns) > len(tc.pipelineruns) {
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

			// Verify Run status contains status for all PipelineRuns.
			_, iterationElements, _ := computeIterations(tc.run, &tc.pipelineloop.Spec)
			expectedPipelineRuns := map[string]pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{}
			i := 1
			for _, pr := range tc.expectedPipelineruns {
				expectedPipelineRuns[pr.Name] = pipelineloopv1alpha1.PipelineLoopPipelineRunStatus{Iteration: i, IterationItem: iterationElements[i-1], Status: &pr.Status}
				if pr.Labels["deleted"] != "True" {
					i = i + 1 // iteration remain same, incase previous pr was a retry.
				}
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
		name:         "invalid range",
		pipelineloop: aPipelineLoop,
		run:          runWithInvalidRange,
		reason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: invalid values for from:-11, to:-13 & step: 1 found in runs`,
		},
	}, {
		name:         "invalid range 2",
		pipelineloop: aPipelineLoop,
		run:          specifyLoopRange("10", "12", "-1", runWithInvalidRange),
		reason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: invalid values for from:10, to:12 & step: -1 found in runs`,
		},
	}, {
		name:         "iterate parameter not an array",
		pipelineloop: aPipelineLoop,
		run:          runWithIterateParamNotAnArray,
		reason:       pipelineloopv1alpha1.PipelineLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: the value of the iterate parameter "current-item" can not transfer to array`,
		},
	}}
	testcases = testcases[len(testcases)-1:]
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

func enableCacheForRun(run *v1alpha1.Run) *v1alpha1.Run {
	run.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] = "true"
	return run
}

func disableCacheForRun(run *v1alpha1.Run) *v1alpha1.Run {
	run.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] = "false"
	return run
}

func enableCacheForPr(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	pr.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] = "true"
	return pr
}

func disableCacheForPr(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	pr.ObjectMeta.Labels["pipelines.kubeflow.org/cache_enabled"] = "false"
	return pr
}

func TestReconcilePipelineLoopRunCachedRun(t *testing.T) {
	testcases := []struct {
		name           string
		pipeline       *v1beta1.Pipeline
		pipelineloop   *pipelineloopv1alpha1.PipelineLoop
		run            *v1alpha1.Run
		pipelineruns   []*v1beta1.PipelineRun
		expectedStatus corev1.ConditionStatus
		expectedReason pipelineloopv1alpha1.PipelineLoopRunReason
		expectedEvents []string
	}{{
		name:           "Reconcile a run successfully",
		pipeline:       aPipeline,
		pipelineloop:   aPipelineLoop,
		run:            enableCacheForRun(loopSucceeded(runPipelineLoop)),
		pipelineruns:   []*v1beta1.PipelineRun{successful(enableCacheForPr(expectedPipelineRunIteration1)), successful(enableCacheForPr(expectedPipelineRunIteration2))},
		expectedStatus: corev1.ConditionTrue,
		expectedReason: pipelineloopv1alpha1.PipelineLoopRunReasonSucceeded,
		expectedEvents: []string{},
	}, {
		name:           "Test fetch from cache for previously successful Run.",
		pipeline:       aPipeline,
		pipelineloop:   aPipelineLoop,
		run:            enableCacheForRun(runPipelineLoop),
		expectedStatus: corev1.ConditionTrue,
		expectedReason: pipelineloopv1alpha1.PipelineLoopRunReasonCacheHit,
		expectedEvents: []string{"Normal Started ", "Normal Succeeded A cached result of the previous run was found."},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()
			optionalPipeline := []*v1beta1.Pipeline{tc.pipeline}
			status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
			tc.pipelineloop.Spec.SetDefaults(ctx)
			status.PipelineLoopSpec = &tc.pipelineloop.Spec
			err := tc.run.Status.EncodeExtraFields(status)
			if err != nil {
				t.Fatal("Failed to encode spec in the pipelineSpec:", err)
			}
			if tc.pipeline == nil {
				optionalPipeline = nil
			}
			cm := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cache-config",
					Namespace: system.Namespace(),
				},
				Data: map[string]string{"driver": "sqlite", "dbName": "/tmp/testing2.db", "timeout": "2s"},
			}
			d := test.Data{
				Runs:         []*v1alpha1.Run{tc.run},
				Pipelines:    optionalPipeline,
				PipelineRuns: tc.pipelineruns,
				ConfigMaps:   []*corev1.ConfigMap{&cm},
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
			// Verify expected events were created.
			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
func checkRunResult(t *testing.T, run *v1alpha1.Run, expectedResult []v1alpha1.RunResult) {
	if len(run.Status.Results) != len(expectedResult) {
		t.Errorf("Expected Run results to include %d results but found %d: %v", len(expectedResult), len(run.Status.Results), run.Status.Results)
		//return
	}

	if d := cmp.Diff(expectedResult, run.Status.Results); d != "" {
		t.Errorf("Run result for is incorrect. Diff %s", diff.PrintWantGot(d))
	}
}

func TestReconcilePipelineLoopRunLastElemResult(t *testing.T) {
	testcases := []struct {
		name           string
		pipeline       *v1beta1.Pipeline
		pipelineloop   *pipelineloopv1alpha1.PipelineLoop
		run            *v1alpha1.Run
		pipelineruns   []*v1beta1.PipelineRun
		expectedResult []v1alpha1.RunResult
	}{{
		name:           "Reconcile a new run with a pipelineloop that references a pipeline",
		pipeline:       aPipeline,
		pipelineloop:   aPipelineLoop,
		run:            disableCacheForRun(runPipelineLoop),
		pipelineruns:   []*v1beta1.PipelineRun{disableCacheForPr(successful(expectedPipelineRunIteration1)), disableCacheForPr(successful(expectedPipelineRunIteration2))},
		expectedResult: []v1alpha1.RunResult{{Name: "last-idx", Value: "2"}, {Name: "last-elem", Value: "item2"}, {Name: "condition", Value: "succeeded"}},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()
			optionalPipeline := []*v1beta1.Pipeline{tc.pipeline}
			status := &pipelineloopv1alpha1.PipelineLoopRunStatus{}
			tc.pipelineloop.Spec.SetDefaults(ctx)
			status.PipelineLoopSpec = &tc.pipelineloop.Spec
			err := tc.run.Status.EncodeExtraFields(status)
			if err != nil {
				t.Fatal("Failed to encode spec in the pipelineSpec:", err)
			}
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
			checkRunResult(t, reconciledRun, tc.expectedResult)
		})
	}
}
