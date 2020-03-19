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

import kfp.dsl as dsl
import json
import os
import shutil
import sys
import tarfile
import tempfile
import unittest
import yaml
import zipfile

from kfp_tekton import compiler

from . import testdata

# after code changes that change the YAML output, temporarily set this flag to True
# in order to generate new "golden" YAML files
GENERATE_GOLDEN_YAML = False


class TestTektonCompiler(unittest.TestCase):

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

  def test_waston_ml_workflow(self):
    """
    Test compiling a Waston ML workflow.
    """
    from .testdata.watson_train_serve_pipeline import kfp_wml_pipeline
    self._test_pipeline_workflow(kfp_wml_pipeline, 'watson_train_serve_pipeline.yaml')

  def _test_pipeline_workflow(self, pipeline_function, pipeline_yaml):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    golden_yaml_file = os.path.join(test_data_dir, pipeline_yaml)
    temp_dir = tempfile.mkdtemp()
    compiled_yaml_file = os.path.join(temp_dir, 'workflow.yaml')
    try:
      compiler.TektonCompiler().compile(pipeline_function, compiled_yaml_file)
      with open(compiled_yaml_file, 'r') as f:
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
