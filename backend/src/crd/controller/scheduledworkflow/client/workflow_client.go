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
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	wraperror "github.com/pkg/errors"
	workflowapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	workflowclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonV1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

// WorkflowClient is a client to call the Workflow API.
type WorkflowClient struct {
	clientSet workflowclientset.Interface
	informer  tektonV1.PipelineRunInformer
}

// NewWorkflowClient creates an instance of the WorkflowClient.
func NewWorkflowClient(clientSet workflowclientset.Interface,
	informer tektonV1.PipelineRunInformer) *WorkflowClient {
	return &WorkflowClient{
		clientSet: clientSet,
		informer:  informer,
	}
}

// AddEventHandler adds an event handler.
func (p *WorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	p.informer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (p *WorkflowClient) HasSynced() func() bool {
	return p.informer.Informer().HasSynced
}

// Get returns a Workflow, given a namespace and name.
func (p *WorkflowClient) Get(namespace string, name string) (
	wf *commonutil.Workflow, isNotFoundError bool, err error) {
	workflow, err := p.informer.Lister().PipelineRuns(namespace).Get(name)
	if err != nil {
		return nil, commonutil.IsNotFound(err), wraperror.Wrapf(err,
			"Error retrieving workflow (%v) in namespace (%v): %v", name, namespace, err)
	}
	return commonutil.NewWorkflow(workflow), false, nil
}

// List returns a list of workflows given the name of their ScheduledWorkflow,
// whether they are completed, and their minimum index (to avoid returning the whole list).
func (p *WorkflowClient) List(swfName string, completed bool, minIndex int64) (
	status []swfapi.WorkflowStatus, err error) {

	labelSelector := getLabelSelectorToGetWorkflows(swfName, completed, minIndex)

	workflows, err := p.informer.Lister().List(*labelSelector)
	if err != nil {
		return nil, wraperror.Wrapf(err,
			"Could not retrieve workflows for scheduled workflow (%v): %v", swfName, err)
	}

	result := toWorkflowStatuses(workflows)

	return result, nil
}

func toWorkflowStatuses(workflows []*workflowapi.PipelineRun) []swfapi.WorkflowStatus {
	result := make([]swfapi.WorkflowStatus, 0)
	for _, workflow := range workflows {
		result = append(result, *toWorkflowStatus(workflow))
	}
	return result
}

func toWorkflowStatus(workflow *workflowapi.PipelineRun) *swfapi.WorkflowStatus {
	return &swfapi.WorkflowStatus{
		Name:        workflow.Name,
		Namespace:   workflow.Namespace,
		SelfLink:    workflow.SelfLink,
		UID:         workflow.UID,
		CreatedAt:   workflow.CreationTimestamp,
		ScheduledAt: retrieveScheduledTime(workflow),
		Index:       retrieveIndex(workflow),
	}
}

func retrieveScheduledTime(workflow *workflowapi.PipelineRun) metav1.Time {
	value, ok := workflow.Labels[commonutil.LabelKeyWorkflowEpoch]
	if !ok {
		return workflow.CreationTimestamp
	}
	result, err := commonutil.RetrieveInt64FromLabel(value)
	if err != nil {
		return workflow.CreationTimestamp
	}
	return metav1.NewTime(time.Unix(result, 0).UTC())
}

func retrieveIndex(workflow *workflowapi.PipelineRun) int64 {
	value, ok := workflow.Labels[commonutil.LabelKeyWorkflowIndex]
	if !ok {
		return 0
	}
	result, err := commonutil.RetrieveInt64FromLabel(value)
	if err != nil {
		return 0
	}
	return result
}

// Create creates a workflow given a namespace and its specification.
func (p *WorkflowClient) Create(ctx context.Context, namespace string, workflow *commonutil.Workflow) (
	*commonutil.Workflow, error) {
	result, err := p.clientSet.TektonV1().PipelineRuns(namespace).Create(ctx, workflow.Get(), metav1.CreateOptions{})
	if err != nil {
		return nil, wraperror.Wrapf(err, "Error creating workflow in namespace (%v): %v: %+v", namespace,
			err, workflow.Get())
	}
	return commonutil.NewWorkflow(result), nil
}

func getLabelSelectorToGetWorkflows(swfName string, completed bool, minIndex int64) *labels.Selector {
	labelSelector := labels.NewSelector()
	// The Argo workflow should be active or completed
	labelSelector = labelSelector.Add(*util.GetRequirementForCompletedWorkflowOrFatal(completed))
	// The Argo workflow should be labelled with this scheduled workflow name.
	labelSelector = labelSelector.Add(*util.GetRequirementForScheduleNameOrFatal(swfName))
	// The Argo workflow should have an index greater than...
	labelSelector = labelSelector.Add(*util.GetRequirementForMinIndexOrFatal(minIndex))
	return &labelSelector
}
