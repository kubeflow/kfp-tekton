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

package main

import (
	"flag"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/reconciler/pipelinelooprun"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

const (
	// ControllerLogKey is the name of the logger for the controller cmd
	ControllerLogKey = "pipelineloop-controller"
)

var (
	namespace = flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")
)

func main() {
	flag.Parse()
	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), *namespace), ControllerLogKey,
		pipelinelooprun.NewController(*namespace),
	)
}
