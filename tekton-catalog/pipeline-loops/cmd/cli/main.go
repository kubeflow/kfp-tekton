/*
Copyright 2021 The Kubeflow Authors

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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/reconciler/pipelinelooprun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func validateFlags(action, inputFileName, inputFileType string) {
	var errs []error
	if action != "validate" {
		err := fmt.Errorf("unsupported action %s\n", action)
		errs = append(errs, err)
	}
	if inputFileName == "" {
		err := fmt.Errorf("missing input spec, please specify -f /path/to/file.(yaml|json)\n")
		errs = append(errs, err)
	}
	if inputFileType != "yaml" && inputFileType != "json" {
		err := fmt.Errorf("unsupported input file type:%s\n", inputFileType)
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		fmt.Println("Errors while processing input:")
		for i, e := range errs {
			fmt.Printf("Error: %d -> %v\n", i, e)
		}
		os.Exit(1)
	}
}

func main() {
	var action, inputFileName, inputFileType string

	flag.StringVar(&action, "a", "validate", "The `action` on the resource.")
	flag.StringVar(&inputFileName, "f", "", "path to input `filename` either a yaml or json.")
	flag.StringVar(&inputFileType, "file-type", "yaml", "`yaml` or a json")
	flag.Parse()
	validateFlags(action, inputFileName, inputFileType)
	if inputFileType == "json" {
		fmt.Println("Error: json file type is not supported.")
		os.Exit(5)
	}
	errs := []string{}
	if inputFileType == "yaml" {
		objs, err := readFile(inputFileName)
		if err != nil {
			fmt.Printf("\nError while reading input spec: %v\n", err)
			os.Exit(2)
		}
		for _, o := range objs {
			marshalledBytes, err := o.MarshalJSON()
			if err != nil {
				fmt.Printf("\nError while marshalling json: %v\n", err)
				os.Exit(3)
			}
			err = nil
			switch kind := o.GetKind(); kind {
			case "Task":
				// No validation.
			case "Run":
				err = validateRun(marshalledBytes)
			case "Pipeline":
				err = validatePipeline(marshalledBytes)
			case "PipelineLoop":
				err = validatePipelineLoop(marshalledBytes)
			case "PipelineRun":
				err = validatePipelineRun(marshalledBytes)
			default:
				fmt.Printf("\nWarn: Unsupported kind: %s.\n", kind)
			}
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			fmt.Printf("\nValidation errors: %s\n", strings.Join(errs, "\n"))
			os.Exit(100)
		} else {
			fmt.Printf("\nCongratulations, all checks passed !!\n")
		}
	}
}

func validatePipeline(bytes []byte) error {
	p := v1beta1.Pipeline{}
	err := json.Unmarshal(bytes, &p)
	if err != nil {
		return err
	}
	return validatePipelineSpec(&p.Spec, p.Name)
}

func validatePipelineSpec(p *v1beta1.PipelineSpec, name string) error {
	errs := []string{}
	// We do not need to validate PipelineRun because it is validated by tekton admission webhook
	// And pipelineTask.TaskRef is also validated by tekton.
	// Here we only need to validate those embedded spec, whose kind is pipelineLoop.
	if p.Tasks != nil {
		for _, task := range p.Tasks {
			if task.TaskSpec != nil && task.TaskSpec.Kind == "PipelineLoop" {
				err := validatePipelineLoopEmbedded(task.TaskSpec.Spec.Raw)
				if err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
	}
	if len(errs) > 0 {
		e := strings.Join(errs, "\n")
		return fmt.Errorf("Validation errors found in pipeline %s\n %s", name, e)
	}
	return nil
}

func validateRun(bytes []byte) error {
	r := v1alpha1.Run{}
	err := json.Unmarshal(bytes, &r)
	if err != nil {
		return err
	}
	// We do not need to validate Run because it is validated by tekton admission webhook
	// And r.Spec.Ref is also validated by tekton.
	// Here we only need to validate the embedded spec. i.e. r.Spec.Spec
	if r.Spec.Spec != nil && r.Spec.Spec.Kind == "PipelineLoop" {
		if err := validatePipelineLoopEmbedded(r.Spec.Spec.Spec.Raw); err != nil {
			return fmt.Errorf("Found validation errors in Run: %s \n %s", r.Name, err.Error())
		}
	}
	return nil
}

func validatePipelineRun(bytes []byte) error {
	pr := v1beta1.PipelineRun{}
	err := json.Unmarshal(bytes, &pr)
	if err != nil {
		return err
	}
	return validatePipelineSpec(pr.Spec.PipelineSpec, pr.Name)
}

func validatePipelineLoopEmbedded(bytes []byte) error {
	var embeddedSpec map[string]interface{}
	if err := json.Unmarshal(bytes, &embeddedSpec); err != nil {
		return err
	}
	r1 := map[string]interface{}{
		"kind":       "PipelineLoop",
		"apiVersion": "custom.tekton.dev/v1alpha1",
		"metadata":   metav1.ObjectMeta{Name: "embedded"},
		"spec":       embeddedSpec,
	}

	marshalBytes, err := json.Marshal(r1)
	if err != nil {
		return err
	}
	return validatePipelineLoop(marshalBytes)
}

func validatePipelineLoop(bytes []byte) error {
	pipelineLoop := pipelineloopv1alpha1.PipelineLoop{}
	if err := json.Unmarshal(bytes, &pipelineLoop); err != nil {
		return err
	}
	ctx := context.Background()
	ctx = pipelinelooprun.EnableCustomTaskFeatureFlag(ctx)
	pipelineLoop.SetDefaults(ctx)
	if err := pipelineLoop.Validate(ctx); err != nil {
		return fmt.Errorf("PipelineLoop name:%s\n %s", pipelineLoop.Name, err.Error())
	}
	if err, name := validateNestedPipelineLoop(pipelineLoop); err != nil {
		return fmt.Errorf("Nested PipelineLoop name:%s\n %s", name, err.Error())
	}
	return nil
}

func validateNestedPipelineLoop(pl pipelineloopv1alpha1.PipelineLoop) (error, string) {
	for _, task := range pl.Spec.PipelineSpec.Tasks {
		if task.TaskSpec != nil && task.TaskSpec.Kind == "PipelineLoop" {
			err := validatePipelineLoopEmbedded(task.TaskSpec.Spec.Raw)
			if err != nil {
				return err, task.Name
			}
		}
	}
	return nil, ""
}

// readFile parses a single file.
func readFile(pathname string) ([]unstructured.Unstructured, error) {
	file, err := os.Open(pathname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return decode(file)
}

// decode consumes the given reader and parses its contents as YAML.
func decode(reader io.Reader) ([]unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLToJSONDecoder(reader)
	objs := []unstructured.Unstructured{}
	var err error
	for {
		out := unstructured.Unstructured{}
		err = decoder.Decode(&out)
		if err != nil {
			break
		}
		if len(out.Object) == 0 {
			continue
		}
		objs = append(objs, out)
	}
	if err != io.EOF {
		return nil, err
	}
	return objs, nil
}
