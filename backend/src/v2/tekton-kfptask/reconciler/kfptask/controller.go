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

package kfptask

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/apis/kfptask"
	kfptaskv1alpha1 "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/apis/kfptask/v1alpha1"
	kfptaskClient "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/client/injection/client"
	kfptaskInformer "github.com/kubeflow/pipelines/backend/src/v2/tekton-kfptask/client/injection/informers/kfptask/v1alpha1/kfptask"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineClient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskRunInformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun"
	customRunInformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/customrun"
	runReconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
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
	run, ok := newObj.(*pipelinev1beta1.CustomRun)
	if ok && (run.IsCancelled() || run.GetDeletionTimestamp() != nil) {
		r.UpdateFunc(oldObj, newObj)
	}
}

// No need to handle delete event
func (r RunEventHandler) OnDelete(obj interface{}) {
}

// A RunEventHandler which only handles Add and specific Update events
type TaskRunEventHandler struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
}

func (r TaskRunEventHandler) OnAdd(obj interface{}) {
	r.AddFunc(obj)
}

// OnUpdate ensures the proper handler is called depending on whether the filter matches
func (r TaskRunEventHandler) OnUpdate(oldObj, newObj interface{}) {
	// only care if the taskrun is done, either succeeded or failed
	run, ok := newObj.(*pipelinev1.TaskRun)
	if ok && run.HasStarted() && (run.IsDone() || run.IsCancelled() || run.IsSuccessful()) {
		r.UpdateFunc(oldObj, newObj)
	}
}

// No need to handle delete event
func (r TaskRunEventHandler) OnDelete(obj interface{}) {
}

// TODO: add caching and logging support
// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		var logger = logging.FromContext(ctx)
		kubeClientSet := kubeClient.Get(ctx)
		pipelineClientSet := pipelineClient.Get(ctx)
		kfptaskClientSet := kfptaskClient.Get(ctx)
		customRunInformer := customRunInformer.Get(ctx)
		kfptaskInformer := kfptaskInformer.Get(ctx)
		taskRunInformer := taskRunInformer.Get(ctx)
		r := &Reconciler{
			kubeClientSet:     kubeClientSet,
			pipelineClientSet: pipelineClientSet,
			kfptaskClientSet:  kfptaskClientSet,
			runLister:         customRunInformer.Lister(),
			kfptaskLister:     kfptaskInformer.Lister(),
			taskRunLister:     taskRunInformer.Lister(),
			clock:             clock.RealClock{},
		}

		impl := runReconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName:     kfptask.ControllerName,
				FinalizerName: kfptask.FinalizerName,
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs of KfpTask
		customRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelineController.FilterCustomRunRef(kfptaskv1alpha1.SchemeGroupVersion.String(), kfptask.Kind),
			Handler: RunEventHandler{
				// Only handle add and update event, because of finalizer, a deletion creates
				// an update event followed by a delete event
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.Enqueue),
			},
		})
		// Add event handler for PipelineRuns controlled by Run
		taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelineController.FilterOwnerCustomRunRef(customRunInformer.Lister(), kfptaskv1alpha1.SchemeGroupVersion.String(), kfptask.Kind),
			Handler: TaskRunEventHandler{
				// Only handle add and update event, because of finalizer, a deletion creates
				// an update event followed by a delete event
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			},
		})

		return impl
	}
}

// For go-clinet 1.27+ in the future
// func composeAddFunc(f func(interface{})) func(interface{}, bool) {
// 	return func(obj interface{}, isInInitialList bool) {
// 		f(obj)
// 	}
// }
