# Copyright 2020 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import shutil
import tempfile
import unittest
import yaml
import re

from kfp_tekton import compiler


# after code changes that change the YAML output, temporarily set this flag to True
# in order to generate new "golden" YAML files
GENERATE_GOLDEN_YAML = False


class TestTektonCompiler(unittest.TestCase):

  def test_init_container_workflow(self):
    """
    Test compiling a initial container workflow.
    """
    from .testdata.init_container import init_container_pipeline
    self._test_pipeline_workflow(init_container_pipeline, 'init_container.yaml')

  def test_sequential_workflow(self):
    """
    Test compiling a sequential workflow.
    """
    from .testdata.sequential import sequential_pipeline
    self._test_pipeline_workflow(sequential_pipeline, 'sequential.yaml')

  def test_parallel_join_workflow(self):
    """
    Test compiling a parallel join workflow.
    """
    from .testdata.parallel_join import download_and_join
    self._test_pipeline_workflow(download_and_join, 'parallel_join.yaml')

  def test_parallel_join_with_argo_vars_workflow(self):
    """
    Test compiling a parallel join workflow.
    """
    from .testdata.parallel_join_with_argo_vars import download_and_join_with_argo_vars
    self._test_pipeline_workflow(download_and_join_with_argo_vars, 'parallel_join_with_argo_vars.yaml')

  def test_pipelinerun_workflow(self):
    """
    Test compiling a parallel join workflow with pipelinerun.
    """
    from .testdata.parallel_join import download_and_join
    self._test_pipeline_workflow(download_and_join, 'parallel_join_pipelinerun.yaml', generate_pipelinerun=True)

  def test_sidecar_workflow(self):
    """
    Test compiling a sidecar workflow.
    """
    from .testdata.sidecar import sidecar_pipeline
    self._test_pipeline_workflow(sidecar_pipeline, 'sidecar.yaml')
  
  def test_loop_static_workflow(self):
    """
    Test compiling a loop static params in workflow.
    """
    from .testdata.loop_static import pipeline
    self._test_pipeline_workflow(
      pipeline,
      'loop_static.yaml',
      normalize_compiler_output_function=lambda f: re.sub(
          "loop-item-param-.*-subvar", "loop-item-param-subvar", f))

  def test_withitem_nested_workflow(self):
    """
    Test compiling a withitem nested in workflow.
    """
    from .testdata.withitem_nested import pipeline
    self._test_pipeline_workflow(pipeline, 'withitem_nested.yaml')

  def test_pipelineparams_workflow(self):
    """
    Test compiling a pipelineparams workflow.
    """
    from .testdata.pipelineparams import pipelineparams_pipeline
    self._test_pipeline_workflow(pipelineparams_pipeline, 'pipelineparams.yaml')

  def test_retry_workflow(self):
    """
    Test compiling a retry task in workflow.
    """
    from .testdata.retry import retry_sample_pipeline
    self._test_pipeline_workflow(retry_sample_pipeline, 'retry.yaml')

  def test_volume_workflow(self):
    """
    Test compiling a volume workflow.
    """
    from .testdata.volume import volume_pipeline
    self._test_pipeline_workflow(volume_pipeline, 'volume.yaml')

  def test_timeout_pipelinerun(self):
    """
    Test compiling a timeout for a whole workflow.
    """
    from .testdata.timeout import timeout_sample_pipeline
    self._test_pipeline_workflow(timeout_sample_pipeline, 'timeout_pipelinerun.yaml', generate_pipelinerun=True)

  def test_timeout_workflow(self):
    """
    Test compiling a step level timeout workflow.
    """
    from .testdata.timeout import timeout_sample_pipeline
    self._test_pipeline_workflow(timeout_sample_pipeline, 'timeout.yaml')

  def test_resourceOp_workflow(self):
    """
    Test compiling a resourceOp basic workflow.
    """
    from .testdata.resourceop_basic import resourceop_basic
    self._test_pipeline_workflow(resourceop_basic, 'resourceop_basic.yaml')

  def test_volumeOp_workflow(self):
    """
    Test compiling a volumeOp basic workflow.
    """
    from .testdata.volume_op import volumeop_basic
    self._test_pipeline_workflow(volumeop_basic, 'volume_op.yaml')

  def test_volumeSnapshotOp_workflow(self):
    """
    Test compiling a volumeSnapshotOp basic workflow.
    """
    from .testdata.volume_snapshot_op import volume_snapshotop_sequential
    self._test_pipeline_workflow(volume_snapshotop_sequential, 'volume_snapshot_op.yaml')

  def test_hidden_output_file_workflow(self):
    """
    Test compiling a workflow with non configurable output file.
    """
    from .testdata.hidden_output_file import hidden_output_file_pipeline
    self._test_pipeline_workflow(hidden_output_file_pipeline, 'hidden_output_file.yaml')

  def test_tolerations_workflow(self):
    """
    Test compiling a tolerations workflow.
    """
    from .testdata.tolerations import tolerations
    self._test_pipeline_workflow(tolerations, 'tolerations.yaml', generate_pipelinerun=True)

  def test_affinity_workflow(self):
    """
    Test compiling a affinity workflow.
    """
    from .testdata.affinity import affinity_pipeline
    self._test_pipeline_workflow(affinity_pipeline, 'affinity.yaml', generate_pipelinerun=True)

  def test_node_selector_workflow(self):
    """
    Test compiling a node selector workflow.
    """
    from .testdata.node_selector import node_selector_pipeline
    self._test_pipeline_workflow(node_selector_pipeline, 'node_selector.yaml', generate_pipelinerun=True)

  def test_pipeline_transformers_workflow(self):
    """
    Test compiling a pipeline_transformers workflow with pod annotations and labels.
    """
    from .testdata.pipeline_transformers import transform_pipeline
    self._test_pipeline_workflow(transform_pipeline, 'pipeline_transformers.yaml')

  def test_artifact_location_workflow(self):
    """
    Test compiling a artifact location workflow.
    """
    from .testdata.artifact_location import custom_artifact_location
    self._test_pipeline_workflow(custom_artifact_location, 'artifact_location.yaml', enable_artifacts=True)

  def test_katib_workflow(self):
    """
    Test compiling a katib workflow.
    """
    from .testdata.katib import mnist_hpo
    self._test_pipeline_workflow(mnist_hpo, 'katib.yaml')
    
  def test_imagepullsecrets_workflow(self):
    """ 
    Test compiling a imagepullsecrets workflow.
    """
    from .testdata.imagepullsecrets import imagepullsecrets_pipeline
    self._test_pipeline_workflow(imagepullsecrets_pipeline, 'imagepullsecrets.yaml', generate_pipelinerun=True)

  def test_basic_no_decorator(self):
    """
    Test compiling a basic workflow with no pipeline decorator
    """
    from .testdata import basic_no_decorator
    parameter_dict = {
      "function": basic_no_decorator.save_most_frequent_word,
      "name": 'Save Most Frequent',
      "description": 'Get Most Frequent Word and Save to GCS',
      "paramsList": [basic_no_decorator.message_param, basic_no_decorator.output_path_param]
    }
    self._test_workflow_without_decorator('basic_no_decorator.yaml', parameter_dict)

  def test_compose(self):
    """
    Test compiling a simple workflow, and a bigger one composed from a simple one.
    """
    from .testdata import compose
    self._test_nested_workflow('compose.yaml', [compose.save_most_frequent_word, compose.download_save_most_frequent_word])
    
  def _test_pipeline_workflow(self, 
                              pipeline_function,
                              pipeline_yaml,
                              generate_pipelinerun=False,
                              enable_artifacts=False,
                              normalize_compiler_output_function=None):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    golden_yaml_file = os.path.join(test_data_dir, pipeline_yaml)
    temp_dir = tempfile.mkdtemp()
    compiled_yaml_file = os.path.join(temp_dir, 'workflow.yaml')
    try:
      compiler.TektonCompiler().compile(pipeline_function,
                                        compiled_yaml_file,
                                        generate_pipelinerun=generate_pipelinerun,
                                        enable_artifacts=enable_artifacts)
      with open(compiled_yaml_file, 'r') as f:
        f = normalize_compiler_output_function(
          f.read()) if normalize_compiler_output_function else f
        compiled = list(yaml.safe_load_all(f))
      if GENERATE_GOLDEN_YAML:
        with open(golden_yaml_file, 'w') as f:
          yaml.dump_all(compiled, f, default_flow_style=False)
      else:
        with open(golden_yaml_file, 'r') as f:
          golden = list(yaml.safe_load_all(f))
        self.maxDiff = None
        self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(temp_dir)

  def _test_workflow_without_decorator(self, pipeline_yaml, params_dict):
    """Test compiling a workflow and appending pipeline params."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    golden_yaml_file = os.path.join(test_data_dir, pipeline_yaml)
    temp_dir = tempfile.mkdtemp()
    try:
      compiled_workflow = compiler.TektonCompiler()._create_workflow(
          params_dict['function'],
          params_dict.get('name', None),
          params_dict.get('description', None),
          params_dict.get('paramsList', None),
          params_dict.get('conf', None))
      if GENERATE_GOLDEN_YAML:
        with open(golden_yaml_file, 'w') as f:
          yaml.dump_all(compiled_workflow, f, default_flow_style=False)
      else:
        with open(golden_yaml_file, 'r') as f:
          golden = list(yaml.safe_load_all(f))
        self.maxDiff = None
        self.assertEqual(golden, compiled_workflow)
    finally:
      shutil.rmtree(temp_dir)

  def _test_nested_workflow(self, 
                            pipeline_yaml,
                            pipeline_list,
                            generate_pipelinerun=False,
                            enable_artifacts=False,
                            normalize_compiler_output_function=None):
    """Test compiling a simple workflow, and a bigger one composed from the simple one."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    golden_yaml_file = os.path.join(test_data_dir, pipeline_yaml)
    temp_dir = tempfile.mkdtemp()
    compiled_yaml_file = os.path.join(temp_dir, 'nested' + str(len(pipeline_list) - 1) + '.yaml')
    try:
      for index, pipeline in enumerate(pipeline_list):
          package_path = os.path.join(temp_dir, 'nested' + str(index) + '.yaml')
          compiler.TektonCompiler().compile(pipeline,
                                            package_path,
                                            generate_pipelinerun=generate_pipelinerun,
                                            enable_artifacts=enable_artifacts)
      with open(compiled_yaml_file, 'r') as f:
        f = normalize_compiler_output_function(
          f.read()) if normalize_compiler_output_function else f
        compiled = list(yaml.safe_load_all(f))
      if GENERATE_GOLDEN_YAML:
        with open(golden_yaml_file, 'w') as f:
          yaml.dump_all(compiled, f, default_flow_style=False)
      else:
        with open(golden_yaml_file, 'r') as f:
          golden = list(yaml.safe_load_all(f))
        self.maxDiff = None
        self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(temp_dir)
