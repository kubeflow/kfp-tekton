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

	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop"
	pipelineloopv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/reconciler/pipelinelooprun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
		fmt.Println("Errors while processing input for filename: ", inputFileName)
		for i, e := range errs {
			fmt.Printf("Error: %d -> %v\n", i, e)
		}
		os.Exit(1)
	}
}

func main() {
	var action, inputFileName, inputFileType string
	var quiet bool
	flag.StringVar(&action, "a", "validate", "The `action` on the resource.")
	flag.StringVar(&inputFileName, "f", "", "path to input `filename` either a yaml or json.")
	flag.StringVar(&inputFileType, "file-type", "yaml", "`yaml` or a json")
	flag.BoolVar(&quiet, "q", false, "quiet mode, report only errors.")
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
			fmt.Printf("\nWarn: skipping file:%s, %v\n", inputFileName, err)
			// os.Exit(2) These are warning because, certain yaml are not k8s resource and we can skip.
			os.Exit(2)
		}
		fmt.Printf("Reading file: %s\n", inputFileName)
		for _, o := range objs {
			marshalledBytes, err := o.MarshalJSON()
			if err != nil {
				fmt.Printf("\nWarn: skipping file due to json Marshal errors:%s, %v\n", inputFileName, err)
				// os.Exit(3) These are warning because, certain yaml are not k8s resource and we can skip.
				os.Exit(0)
			}
			err = nil
			switch kind := o.GetKind(); kind {
			case "Task":
				// No validation.
			case "Run":
				err = validateRun(marshalledBytes)
			case "CustomRun":
				err = validateCustomRun(marshalledBytes)
			case "Pipeline":
				err = validatePipeline(marshalledBytes)
			case pipelineloop.PipelineLoopControllerName:
				err = validatePipelineLoop(marshalledBytes)
			case "PipelineRun":
				err = validatePipelineRun(marshalledBytes)
			default:
				if !quiet {
					fmt.Printf("\nWarn: Unsupported kind: %s.\n", kind)
				}
			}
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			fmt.Printf("\nFound validation errors in %s: \n%s\n", inputFileName,
				strings.Join(errs, "\n"))
			os.Exit(100)
		} else {
			if !quiet {
				fmt.Printf("\nCongratulations, all checks passed in %s\n", inputFileName)
			}
		}
	}
}

func validatePipeline(bytes []byte) error {
	p := v1.Pipeline{}
	if err := json.Unmarshal(bytes, &p); err != nil {
		return err
	}
	return validatePipelineSpec(&p.Spec, p.Name)
}

func validatePipelineSpec(p *v1.PipelineSpec, name string) error {
	errs := []string{}
	// Here we validate those embedded spec, whose kind is pipelineLoop.
	if p.Tasks != nil {
		for _, task := range p.Tasks {
			if task.TaskSpec != nil && task.TaskSpec.Kind == pipelineloop.PipelineLoopControllerName {
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

	// Validate the Tekton pipelineSpec
	ctx := context.Background()
	ctx = pipelinelooprun.EnableCustomTaskFeatureFlag(ctx)
	p.SetDefaults(ctx)
	errors := p.Validate(ctx)
	if errors != nil {
		return errors
	}
	return nil
}

func validateRun(bytes []byte) error {
	r := v1alpha1.Run{}
	err := json.Unmarshal(bytes, &r)
	if err != nil {
		return fmt.Errorf("Error while unmarshal Run:%s\n", err.Error())
	}
	// We do not need to validate Run because it is validated by tekton admission webhook
	// And r.Spec.Ref is also validated by tekton.
	// Here we only need to validate the embedded spec. i.e. r.Spec.Spec
	if r.Spec.Spec != nil && r.Spec.Spec.Kind == pipelineloop.PipelineLoopControllerName {
		if err := validatePipelineLoopEmbedded(r.Spec.Spec.Spec.Raw); err != nil {
			return fmt.Errorf("Found validation errors in Run: %s \n %s", r.Name, err.Error())
		}
	}
	return nil
}

func validateCustomRun(bytes []byte) error {
	customRun := v1beta1.CustomRun{}
	err := json.Unmarshal(bytes, &customRun)
	if err != nil {
		return fmt.Errorf("Error while unmarshal CustomRun:%s\n", err.Error())
	}
	// We do not need to validate CustomRun because it is validated by tekton admission webhook
	// And r.Spec.CustomRef is also validated by tekton.
	// Here we only need to validate the embedded spec. i.e. r.Spec.Spec
	if customRun.Spec.CustomSpec != nil && customRun.Spec.CustomSpec.Kind == pipelineloop.PipelineLoopControllerName {
		if err := validatePipelineLoopEmbedded(customRun.Spec.CustomSpec.Spec.Raw); err != nil {
			return fmt.Errorf("Found validation errors in CustomRun: %s \n %s", customRun.Name, err.Error())
		}
	}
	return nil
}

func validatePipelineRun(bytes []byte) error {
	pr := v1.PipelineRun{}
	if err := json.Unmarshal(bytes, &pr); err != nil {
		return fmt.Errorf("Error while unmarshal PipelineRun spec:%s\n", err.Error())
	}
	return validatePipelineSpec(pr.Spec.PipelineSpec, pr.Name)
}

func validatePipelineLoopEmbedded(bytes []byte) error {
	var embeddedSpec map[string]interface{}
	if err := json.Unmarshal(bytes, &embeddedSpec); err != nil {
		return fmt.Errorf("Error while unmarshal PipelineLoop embedded spec:%s\n", err.Error())
	}
	r1 := map[string]interface{}{
		"kind":       pipelineloop.PipelineLoopControllerName,
		"apiVersion": pipelineloopv1alpha1.SchemeGroupVersion.String(),
		"metadata":   metav1.ObjectMeta{Name: "embedded"},
		"spec":       embeddedSpec,
	}

	marshalBytes, err := json.Marshal(r1)
	if err != nil {
		return fmt.Errorf("Error while marshalling embedded to PipelineLoop:%s\n", err.Error())
	}
	return validatePipelineLoop(marshalBytes)
}

func validatePipelineLoop(bytes []byte) error {
	pipelineLoop := pipelineloopv1alpha1.PipelineLoop{}
	if err := json.Unmarshal(bytes, &pipelineLoop); err != nil {
		return fmt.Errorf("Error while unmarshal PipelineLoop:%s\n", err.Error())
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
	if pl.Spec.PipelineSpec != nil {
		for _, task := range pl.Spec.PipelineSpec.Tasks {
			if task.TaskSpec != nil && task.TaskSpec.Kind == pipelineloop.PipelineLoopControllerName {
				err := validatePipelineLoopEmbedded(task.TaskSpec.Spec.Raw)
				if err != nil {
					return err, task.Name
				}
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
