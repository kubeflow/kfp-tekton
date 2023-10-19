package controller

import (
	context "context"

	runInformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/customrun"
	customrun "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "kfp-driver"
	apiVersion     = "kfp-driver.tekton.dev/v1alpha1"
	kind           = "KFPDriver"
)

// NewController creates a Reconciler for Run and returns the result of NewImpl.
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	runInformer := runInformer.Get(ctx)

	r := &Reconciler{}

	impl := customrun.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: ControllerName,
		}
	})

	logger.Info("Setting up event handlers")

	runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: tkncontroller.FilterCustomRunRef(apiVersion, kind),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	return impl
}
