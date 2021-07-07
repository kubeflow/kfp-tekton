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

package v1alpha1

import (
	"context"
	"os"
	"strconv"

	"knative.dev/pkg/apis"
)

var _ apis.Defaultable = (*PipelineLoop)(nil)

// SetDefaults set any defaults for the PipelineLoop
func (tl *PipelineLoop) SetDefaults(ctx context.Context) {
	tl.Spec.SetDefaults(ctx)
}

// SetDefaults set any defaults for the PipelineLoop spec
func (tls *PipelineLoopSpec) SetDefaults(ctx context.Context) {
	if tls.Parallelism == 0 {
		parallelism := os.Getenv("LOOP_PARALLELISM")
		if parallelism == "" {
			tls.Parallelism = 1
			return
		}
		i, err := strconv.Atoi(parallelism)
		if err != nil {
			// fall back to default 1
			tls.Parallelism = 1
			return
		} else {
			tls.Parallelism = i
		}
	}
}
