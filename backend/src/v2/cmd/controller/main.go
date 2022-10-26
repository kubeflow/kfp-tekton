package main

import (
	"github.com/kubeflow/pipelines/backend/src/v2/controller"
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	sharedmain.Main(controller.ControllerName, controller.NewController)
}
