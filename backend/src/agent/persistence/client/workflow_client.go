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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	wfapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	wfclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// WorkflowClient is a client to call the Workflow API.
type WorkflowClient struct {
	informer  v1beta1.PipelineRunInformer
	clientset *wfclientset.Clientset
}

// NewWorkflowClient creates an instance of the WorkflowClient.
func NewWorkflowClient(informer v1beta1.PipelineRunInformer,
	clientset *wfclientset.Clientset) *WorkflowClient {

	return &WorkflowClient{
		informer:  informer,
		clientset: clientset,
	}
}

// AddEventHandler adds an event handler.
func (c *WorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	c.informer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (c *WorkflowClient) HasSynced() func() bool {
	return c.informer.Informer().HasSynced
}

// Get returns a Workflow, given a namespace and name.
func (c *WorkflowClient) Get(namespace string, name string) (
	wf *util.Workflow, err error) {
	workflow, err := c.informer.Lister().PipelineRuns(namespace).Get(name)
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
	if err := c.getStatusFromChildReferences(namespace,
		fmt.Sprintf("%s=%s", util.LabelKeyWorkflowRunId, workflow.Labels[util.LabelKeyWorkflowRunId]),
		&workflow.Status); err != nil {

		return nil, err
	}

	return util.NewWorkflow(workflow), nil
}

func (c *WorkflowClient) getStatusFromChildReferences(namespace, selector string, status *wfapi.PipelineRunStatus) error {
	if status.ChildReferences != nil {
		hasTaskRun, hasRun := false, false
		for _, child := range status.ChildReferences {
			switch child.Kind {
			case "TaskRun":
				hasTaskRun = true
			case "Run":
				hasRun = true
			default:
			}
		}
		// TODO: restruct the workflow to contain taskrun/run status, these 2 field
		// will be removed in the future
		if hasTaskRun {
			// fetch taskrun status and insert into Status.TaskRuns
			taskruns, err := c.clientset.TektonV1beta1().TaskRuns(namespace).List(context.Background(), v1.ListOptions{
				LabelSelector: selector,
			})
			if err != nil {
				return util.NewInternalServerError(err, "can't fetch taskruns")
			}

			taskrunStatuses := make(map[string]*wfapi.PipelineRunTaskRunStatus, len(taskruns.Items))
			for _, taskrun := range taskruns.Items {
				taskrunStatuses[taskrun.Name] = &wfapi.PipelineRunTaskRunStatus{
					PipelineTaskName: taskrun.Labels["tekton.dev/pipelineTask"],
					Status:           taskrun.Status.DeepCopy(),
				}
			}
			status.TaskRuns = taskrunStatuses
		}
		if hasRun {
			runs, err := c.clientset.TektonV1alpha1().Runs(namespace).List(context.Background(), v1.ListOptions{
				LabelSelector: selector,
			})
			if err != nil {
				return util.NewInternalServerError(err, "can't fetch runs")
			}
			runStatuses := make(map[string]*wfapi.PipelineRunRunStatus, len(runs.Items))
			for _, run := range runs.Items {
				runStatus := run.Status.DeepCopy()
				runStatuses[run.Name] = &wfapi.PipelineRunRunStatus{
					PipelineTaskName: run.Labels["tekton.dev/pipelineTask"],
					Status:           runStatus,
				}
				// handle nested status
				c.handleNestedStatus(&run, runStatus, namespace)
			}
			status.Runs = runStatuses
		}
	}
	return nil
}

// handle nested status case for specific types of Run
func (c *WorkflowClient) handleNestedStatus(run *v1alpha1.Run, runStatus *v1alpha1.RunStatus, namespace string) {
	if sort.SearchStrings(childReferencesKinds, run.Spec.Spec.Kind) < len(childReferencesKinds) {
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
						if taskrunCR, err := c.clientset.TektonV1beta1().TaskRuns(namespace).Get(context.Background(), name, v1.GetOptions{}); err == nil {
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
					} else if kind == "Run" {
						if runCR, err := c.clientset.TektonV1alpha1().Runs(namespace).Get(context.Background(), name, v1.GetOptions{}); err == nil {
							runs, ok := statusobj["runs"]
							if !ok {
								runs = make(map[string]*wfapi.PipelineRunRunStatus)
							}
							if runStatus, ok := runs.(map[string]*wfapi.PipelineRunRunStatus); ok {
								runStatusStatus := runCR.Status.DeepCopy()
								runStatus[name] = &wfapi.PipelineRunRunStatus{
									PipelineTaskName: runCR.Labels["tekton.dev/pipelineTask"],
									Status:           runStatusStatus,
								}
								statusobj["runs"] = runStatus
								// handle nested status recursively
								c.handleNestedStatus(runCR, runStatusStatus, namespace)
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
