// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tektoncompiler_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/tektoncompiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
)

var update = flag.Bool("update", true, "update golden files")

func Test_tekton_compiler(t *testing.T) {
	tests := []struct {
		jobPath          string // path of input PipelineJob to compile
		platformSpecPath string // path of platform spec
		tektonYAMLPath   string // path of expected output argo workflow YAML
	}{
		{
			jobPath:          "../testdata/hello_world.json",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/hello_world.yaml",
		},
		{
			jobPath:          "../testdata/importer.json",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/importer.yaml",
		},
		{
			jobPath:          "../testdata/importer.json",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/importer.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {

			job, platformSpec := load(t, tt.jobPath, tt.platformSpecPath, "json")
			if *update {
				pr, err := tektoncompiler.Compile(job, platformSpec, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := yaml.Marshal(pr)
				if err != nil {
					t.Fatal(err)
				}
				err = ioutil.WriteFile(tt.tektonYAMLPath, got, 0x664)
				if err != nil {
					t.Fatal(err)
				}
			}
			tektonYAML, err := ioutil.ReadFile(tt.tektonYAMLPath)
			if err != nil {
				t.Fatal(err)
			}
			pr, err := tektoncompiler.Compile(job, platformSpec, nil)
			if err != nil {
				t.Error(err)
			}
			var expected pipelineapi.PipelineRun
			err = yaml.Unmarshal(tektonYAML, &expected)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(pr, &expected) {
				t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n", tt.jobPath, cmp.Diff(&expected, pr))
			}
		})

	}

}

func TestMnist(t *testing.T) {
	tests := []struct {
		yamlPath         string // path of input PipelineJob to compile
		platformSpecPath string // path of platform spec
		tektonYAMLPath   string // path of expected output argo workflow YAML
	}{
		{
			yamlPath:         "testdata/mnist_pipeline_ir.yaml",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/mnist_pipeline.yaml",
		},
		{
			yamlPath:         "testdata/exit_handler_ir.yaml",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/exit_handler.yaml",
		},
		{
			yamlPath:         "testdata/loop_static_ir.yaml",
			platformSpecPath: "",
			tektonYAMLPath:   "testdata/loop_static.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {

			job, platformSpec := load(t, tt.yamlPath, tt.platformSpecPath, "yaml")
			if *update {
				pr, err := tektoncompiler.Compile(job, platformSpec, nil)
				if err != nil {
					t.Fatal(err)
				}
				got, err := yaml.Marshal(pr)
				if err != nil {
					t.Fatal(err)
				}
				err = ioutil.WriteFile(tt.tektonYAMLPath, got, 0644)
				if err != nil {
					t.Fatal(err)
				}
			}
			tektonYAML, err := ioutil.ReadFile(tt.tektonYAMLPath)
			if err != nil {
				t.Fatal(err)
			}
			pr, err := tektoncompiler.Compile(job, platformSpec, nil)
			if err != nil {
				t.Error(err)
			}
			var expected pipelineapi.PipelineRun
			err = yaml.Unmarshal(tektonYAML, &expected)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(pr, &expected) {
				t.Errorf("compiler.Compile(%s)!=expected, diff: %s\n", tt.yamlPath, cmp.Diff(pr, &expected))
			}
		})

	}
}

func load(t *testing.T, path string, platformSpecPath string, fileType string) (*pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec) {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	if fileType == "yaml" {
		content, err = yaml.YAMLToJSON(content)
		if err != nil {
			t.Error(err)
		}
	}
	job := &pipelinespec.PipelineJob{}
	if err := protojson.Unmarshal(content, job); err != nil {
		t.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, string(content))
	}

	platformSpec := &pipelinespec.PlatformSpec{}
	if platformSpecPath != "" {
		content, err = ioutil.ReadFile(platformSpecPath)
		if err != nil {
			t.Error(err)
		}
		if fileType == "yaml" {
			content, err = yaml.YAMLToJSON(content)
			if err != nil {
				t.Error(err)
			}
		}
		if err := protojson.Unmarshal(content, platformSpec); err != nil {
			t.Errorf("Failed to parse platform spec, error: %s, spec: %v", err, string(content))
		}
		return job, platformSpec.Platforms["kubernetes"]
	}
	return job, nil
}
