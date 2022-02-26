package resource

import "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

func AddRuntimeMetadata(wf *v1beta1.PipelineRun) {
	wf.Annotations = map[string]string{"sidecar.istio.io/inject": "false"}
	wf.Labels = map[string]string{"pipelines.kubeflow.org/cache_enabled": "true"}
}
