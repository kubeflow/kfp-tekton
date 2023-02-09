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

// Code generated by injection-gen. DO NOT EDIT.

package fakeFilteredFactory

import (
	context "context"

	externalversions "github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client/informers/externalversions"
	fake "github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client/injection/client/fake"
	filtered "github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client/injection/informers/factory/filtered"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

var Get = filtered.Get

func init() {
	injection.Fake.RegisterInformerFactory(withInformerFactory)
}

func withInformerFactory(ctx context.Context) context.Context {
	c := fake.Get(ctx)
	untyped := ctx.Value(filtered.LabelKey{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch labelkey from context.")
	}
	labelSelectors := untyped.([]string)
	for _, selector := range labelSelectors {
		opts := []externalversions.SharedInformerOption{}
		if injection.HasNamespaceScope(ctx) {
			opts = append(opts, externalversions.WithNamespace(injection.GetNamespaceScope(ctx)))
		}
		opts = append(opts, externalversions.WithTweakListOptions(func(l *v1.ListOptions) {
			l.LabelSelector = selector
		}))
		ctx = context.WithValue(ctx, filtered.Key{Selector: selector},
			externalversions.NewSharedInformerFactoryWithOptions(c, controller.GetResyncPeriod(ctx), opts...))
	}
	return ctx
}
