// Copyright 2018 The Kubeflow Authors
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

package client

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	wfapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	wfapiV1Beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	customRun "github.com/tektoncd/pipeline/pkg/apis/run/v1beta1"
	wfclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonV1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1"
	tektonV1Beta1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type runKinds []string

var (
	// A list of Kinds that contains childReferences
	// those childReferences would be scaned and retrieve their taskrun/run status
	childReferencesKinds runKinds
)

const (
	childReferencesKindFlagName = "childReferencesKinds"
)

func (rk *runKinds) String() string {
	return fmt.Sprint(*rk)
}

func (rk *runKinds) Set(value string) error {
	if len(*rk) > 0 {
		return fmt.Errorf("%s has been set", childReferencesKindFlagName)
	}

	for _, k := range strings.Split(value, ",") {
		*rk = append(*rk, k)
	}
	sort.Strings(*rk)

	return nil
}

func init() {
	flag.Var(&childReferencesKinds, childReferencesKindFlagName, "A list of kinds to search for the nested childReferences")
}

type WorkflowClientInterface interface {
	Get(namespace string, name string) (wf *util.Workflow, err error)
}

type Informers struct {
	TRInformer tektonV1.TaskRunInformer
	PRInformer tektonV1.PipelineRunInformer
	CRInformer tektonV1Beta1.CustomRunInformer
}

// WorkflowClient is a client to call the Workflow API.
type WorkflowClient struct {
	informers Informers
	clientset *wfclientset.Clientset
}

// NewWorkflowClient creates an instance of the WorkflowClient.
func NewWorkflowClient(informers Informers,
	clientset *wfclientset.Clientset) *WorkflowClient {

	return &WorkflowClient{
		informers: informers,
		clientset: clientset,
	}
}

// AddEventHandler adds an event handler.
func (c *WorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	c.informers.PRInformer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (c *WorkflowClient) HasSynced() func() bool {
	return func() bool {
		log.Infof("cache sync: PR: %t, TR: %t, CR: %t", c.informers.PRInformer.Informer().HasSynced(),
			c.informers.TRInformer.Informer().HasSynced(), c.informers.TRInformer.Informer().HasSynced())
		return c.informers.PRInformer.Informer().HasSynced() &&
			c.informers.TRInformer.Informer().HasSynced() && c.informers.TRInformer.Informer().HasSynced()
	}
}

// Get returns a Workflow, given a namespace and name.
func (c *WorkflowClient) Get(namespace string, name string) (
	wf *util.Workflow, err error) {
	workflow, err := c.informers.PRInformer.Lister().PipelineRuns(namespace).Get(name)
	if err != nil {
		var code util.CustomCode
		if util.IsNotFound(err) {
			code = util.CUSTOM_CODE_NOT_FOUND
		} else {
			code = util.CUSTOM_CODE_GENERIC
		}
		return nil, util.NewCustomError(err, code,
			"Error retrieving workflow (%v) in namespace (%v): %v", name, namespace, err)
	}

	s, err := labels.ConvertSelectorToLabelsMap(fmt.Sprintf("%s=%s", util.LabelKeyWorkflowRunId, workflow.Labels[util.LabelKeyWorkflowRunId]))
	if err != nil {
		return nil, err
	}

	if err := c.getStatusFromChildReferences(namespace, labels.SelectorFromSet(s), &workflow.Status, workflow.Labels); err != nil {
		return nil, err
	}

	return util.NewWorkflow(workflow), nil
}

func (c *WorkflowClient) getStatusFromChildReferences(namespace string, selector labels.Selector, status *wfapi.PipelineRunStatus, labels map[string]string) error {
	if status.ChildReferences != nil {
		hasTaskRun, hasCustomRun := false, false
		for _, child := range status.ChildReferences {
			switch child.Kind {
			case "TaskRun":
				hasTaskRun = true
			case "CustomRun":
				hasCustomRun = true
			default:
			}
		}

		// TODO: restruct the workflow to contain taskrun/run status, these 2 field
		// will be removed in the future
		if hasTaskRun {
			// fetch taskrun status and insert into Status.TaskRuns
			taskruns, err := c.informers.TRInformer.Lister().TaskRuns(namespace).List(selector)
			if err != nil {
				return util.NewInternalServerError(err, "can't fetch taskruns")
			}

			taskrunStatuses := make(map[string]*wfapi.PipelineRunTaskRunStatus, len(taskruns))
			for _, taskrun := range taskruns {
				log.Infof("handle childreference(TaskRun): %s", taskrun.Name)
				taskrunStatuses[taskrun.Name] = &wfapi.PipelineRunTaskRunStatus{
					PipelineTaskName: taskrun.Labels["tekton.dev/pipelineTask"],
					Status:           taskrun.Status.DeepCopy(),
				}
			}
			taskrunStatusesMarshal, err := json.Marshal(taskrunStatuses)
			labels["taskrunStatuses"] = string(taskrunStatusesMarshal)
		}
		if hasCustomRun {
			customRuns, err := c.informers.CRInformer.Lister().CustomRuns(namespace).List(selector)
			if err != nil {
				return util.NewInternalServerError(err, "can't fetch runs")
			}
			customRunStatuses := make(map[string]*wfapi.PipelineRunRunStatus, len(customRuns))
			for _, customRun := range customRuns {
				log.Infof("handle childreference(CustomRun): %s", customRun.Name)
				customRunStatus := customRun.Status.DeepCopy()
				customRunStatuses[customRun.Name] = &wfapi.PipelineRunRunStatus{
					PipelineTaskName: customRun.Labels["tekton.dev/pipelineTask"],
					Status:           customRunStatus,
				}
				// handle nested status
				c.handleNestedStatusV1beta1(customRun, customRunStatus, namespace)
			}
			customRunStatusesMarshal, err := json.Marshal(customRunStatuses)
			labels["customRunStatuses"] = string(customRunStatusesMarshal)
		}
	}
	return nil
}

// handle nested status case for specific types of Run
func (c *WorkflowClient) handleNestedStatus(run *v1alpha1.Run, runStatus *v1alpha1.RunStatus, namespace string) {
	var kind string
	if run.Spec.Spec != nil {
		kind = run.Spec.Spec.Kind
	} else if run.Spec.Ref != nil {
		kind = string(run.Spec.Ref.Kind)
	}
	if sort.SearchStrings(childReferencesKinds, kind) >= len(childReferencesKinds) {
		return
	}

	// need to lookup the nested status
	obj := make(map[string]interface{})
	if err := json.Unmarshal(runStatus.ExtraFields.Raw, &obj); err != nil {
		return
	}
	if c.updateExtraFields(obj, namespace) {
		if newStatus, err := json.Marshal(obj); err == nil {
			runStatus.ExtraFields.Raw = newStatus
		}
	}
}

// handle nested status case for specific types of Run
func (c *WorkflowClient) handleNestedStatusV1beta1(customRun *wfapiV1Beta1.CustomRun, customRunStatus *customRun.CustomRunStatus, namespace string) {
	var kind string
	if customRun.Spec.CustomSpec != nil {
		kind = customRun.Spec.CustomSpec.Kind
	} else if customRun.Spec.CustomRef != nil {
		kind = string(customRun.Spec.CustomRef.Kind)
	}
	if sort.SearchStrings(childReferencesKinds, kind) >= len(childReferencesKinds) {
		return
	}

	// need to lookup the nested status
	obj := make(map[string]interface{})
	if err := json.Unmarshal(customRunStatus.ExtraFields.Raw, &obj); err != nil {
		return
	}
	if c.updateExtraFields(obj, namespace) {
		if newStatus, err := json.Marshal(obj); err == nil {
			customRunStatus.ExtraFields.Raw = newStatus
		}
	}

}

// check ExtraFields and update nested status if needed
func (c *WorkflowClient) updateExtraFields(obj map[string]interface{}, namespace string) bool {
	updated := false
	val, ok := obj["pipelineRuns"]
	if !ok {
		return false
	}
	prs, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	// go through the list of pipelineRuns
	for _, val := range prs {
		probj, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		val, ok := probj["status"]
		if !ok {
			continue
		}
		statusobj, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		childRef, ok := statusobj["childReferences"]
		if !ok {
			continue
		}
		if children, ok := childRef.([]interface{}); ok {
			for _, childObj := range children {
				if child, ok := childObj.(map[string]interface{}); ok {
					kindI, ok := child["kind"]
					if !ok {
						continue
					}
					nameI, ok := child["name"]
					if !ok {
						continue
					}
					kind := fmt.Sprintf("%v", kindI)
					name := fmt.Sprintf("%v", nameI)
					if kind == "TaskRun" {

						if taskrunCR, err := c.informers.TRInformer.Lister().TaskRuns(namespace).Get(name); err == nil {
							taskruns, ok := statusobj["taskRuns"]
							if !ok {
								taskruns = make(map[string]*wfapi.PipelineRunTaskRunStatus)
							}
							if taskrunStatus, ok := taskruns.(map[string]*wfapi.PipelineRunTaskRunStatus); ok {
								taskrunStatus[name] = &wfapi.PipelineRunTaskRunStatus{
									PipelineTaskName: taskrunCR.Labels["tekton.dev/pipelineTask"],
									Status:           taskrunCR.Status.DeepCopy(),
								}
								statusobj["taskRuns"] = taskrunStatus
								updated = true
							}
						}
					} else if kind == "CustomRun" {
						if customRunsCR, err := c.informers.CRInformer.Lister().CustomRuns(namespace).Get(name); err == nil {
							// still using "runs", cause there is no customRun in status object
							customRuns, ok := statusobj["runs"]
							if !ok {
								customRuns = make(map[string]*wfapi.PipelineRunRunStatus)
							}
							if customRunStatus, ok := customRuns.(map[string]*wfapi.PipelineRunRunStatus); ok {
								customRunStatusStatus := customRunsCR.Status.DeepCopy()
								customRunStatus[name] = &wfapi.PipelineRunRunStatus{
									PipelineTaskName: customRunsCR.Labels["tekton.dev/pipelineTask"],
									Status:           customRunStatusStatus,
								}
								statusobj["runs"] = customRunStatus
								// handle nested status recursively
								c.handleNestedStatusV1beta1(customRunsCR, customRunStatusStatus, namespace)
								updated = true
							}
						}
					}
				}
			}
		}

	}
	return updated
}

// TODO: update status to v1beta1 base once pipelineloop supports v1beta1 customrun
// FromRunStatus converts a *v1alpha1.RunStatus into a corresponding *v1beta1.CustomRunStatus
func FromRunStatus(orig *v1alpha1.RunStatus) *customRun.CustomRunStatus {
	crs := customRun.CustomRunStatus{
		Status: orig.Status,
		CustomRunStatusFields: customRun.CustomRunStatusFields{
			StartTime:      orig.StartTime,
			CompletionTime: orig.CompletionTime,
			ExtraFields:    orig.ExtraFields,
		},
	}

	for _, origRes := range orig.Results {
		crs.Results = append(crs.Results, customRun.CustomRunResult{
			Name:  origRes.Name,
			Value: origRes.Value,
		})
	}

	for _, origRetryStatus := range orig.RetriesStatus {
		crs.RetriesStatus = append(crs.RetriesStatus, customRun.FromRunStatus(origRetryStatus))
	}

	return &crs
}
