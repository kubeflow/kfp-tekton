/*
// Copyright 2023 kubeflow.org
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

package exithandler

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis/exithandler"
	exithandlerv1alpha1 "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis/exithandler/v1alpha1"
	exithandlerClient "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client/injection/client"
	exithandlerInformer "github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client/injection/informers/exithandler/v1alpha1/exithandler"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineClient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	runInformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	pipelineRunInformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	runReconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelineController "github.com/tektoncd/pipeline/pkg/controller"
	"knative.dev/pkg/logging"

	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"
	kubeClient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// A RunEventHandler which only handles Add and specific Update events
type RunEventHandler struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
}

func (r RunEventHandler) OnAdd(obj interface{}) {
	r.AddFunc(obj)
}

// OnUpdate ensures the proper handler is called depending on whether the filter matches
func (r RunEventHandler) OnUpdate(oldObj, newObj interface{}) {
	// handle update event triggered by deletion or cancelation
	run, ok := newObj.(*pipelinev1alpha1.Run)
	if ok && (run.IsCancelled() || run.GetDeletionTimestamp() != nil) {
		r.UpdateFunc(oldObj, newObj)
	}
}

// No need to handle delete event
func (r RunEventHandler) OnDelete(obj interface{}) {
}

// A RunEventHandler which only handles Add and specific Update events
type PipelineRunEventHandler struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
}

func (r PipelineRunEventHandler) OnAdd(obj interface{}) {
	r.AddFunc(obj)
}

// OnUpdate ensures the proper handler is called depending on whether the filter matches
func (r PipelineRunEventHandler) OnUpdate(oldObj, newObj interface{}) {
	// only care if the pipelinerun is done, either succeeded or failed
	run, ok := newObj.(*pipelinev1beta1.PipelineRun)
	if ok && run.HasStarted() && (run.IsDone() || run.IsCancelled() || run.IsGracefullyCancelled() || run.IsGracefullyStopped()) {
		r.UpdateFunc(oldObj, newObj)
	}
}

// No need to handle delete event
func (r PipelineRunEventHandler) OnDelete(obj interface{}) {
}

// TODO: add caching and logging support
// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		var logger = logging.FromContext(ctx)
		kubeClientSet := kubeClient.Get(ctx)
		pipelineClientSet := pipelineClient.Get(ctx)
		exitHandlerClientSet := exithandlerClient.Get(ctx)
		runInformer := runInformer.Get(ctx)
		exitHandlerInformer := exithandlerInformer.Get(ctx)
		pipelineRunInformer := pipelineRunInformer.Get(ctx)
		r := &Reconciler{
			kubeClientSet:        kubeClientSet,
			pipelineClientSet:    pipelineClientSet,
			exitHandlerClientSet: exitHandlerClientSet,
			runLister:            runInformer.Lister(),
			exitHandlerLister:    exitHandlerInformer.Lister(),
			pipelineRunLister:    pipelineRunInformer.Lister(),
			clock:                clock.RealClock{},
		}

		impl := runReconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName:     exithandler.ControllerName,
				FinalizerName: exithandler.FinalizerName,
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs of ExitHandler
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelineController.FilterRunRef(exithandlerv1alpha1.SchemeGroupVersion.String(), exithandler.Kind),
			Handler: RunEventHandler{
				// Only handle add and update event, because of finalizer, a deletion creates
				// an update event followed by a delete event
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.Enqueue),
			},
		})
		// Add event handler for PipelineRuns controlled by Run
		pipelineRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelineController.FilterOwnerRunRef(runInformer.Lister(), exithandlerv1alpha1.SchemeGroupVersion.String(), exithandler.Kind),
			Handler: PipelineRunEventHandler{
				// Only handle add and update event, because of finalizer, a deletion creates
				// an update event followed by a delete event
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			},
		})

		return impl
	}
}
