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

	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	pipelineloopclient "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/client"
	pipelineloopinformer "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/client/injection/informers/pipelineloop/v1alpha1/pipelineloop"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {

		logger := logging.FromContext(ctx)
		pipelineclientset := pipelineclient.Get(ctx)
		pipelineloopclientset := pipelineloopclient.Get(ctx)
		runInformer := runinformer.Get(ctx)
		pipelineLoopInformer := pipelineloopinformer.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		c := &Reconciler{
			pipelineClientSet: pipelineclientset,
			pipelineloopClientSet: pipelineloopclientset,
			runLister:         runInformer.Lister(),
			pipelineLoopLister:    pipelineLoopInformer.Lister(),
			pipelineRunLister: pipelineRunInformer.Lister(),
		}

		impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "run-pipelineloop",
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.PipelineLoopControllerName),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		// Add event handler for PipelineRuns controlled by Run
		pipelineRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterOwnerRunRef(runInformer.Lister(), pipelineloopv1alpha1.SchemeGroupVersion.String(), pipelineloop.PipelineLoopControllerName),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
