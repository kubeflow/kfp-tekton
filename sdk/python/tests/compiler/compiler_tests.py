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


class TestTektonCompiler(unittest.TestCase):

  def _get_yaml_from_zip(self, zip_file):
    with zipfile.ZipFile(zip_file, 'r') as zip:
      return yaml.safe_load(zip.read(zip.namelist()[0]))

  def _get_yaml_from_tar(self, tar_file):
    with tarfile.open(tar_file, 'r:gz') as tar:
      return yaml.safe_load(tar.extractfile(tar.getmembers()[0]))

  def test_sequential_workflow(self):
    """
    Test compiling a sequential workflow.
    """
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)

    from .testdata import sequential
    tmpdir = tempfile.mkdtemp()
    package_path = os.path.join(tmpdir, 'workflow.zip')
    try:
      compiler.TektonCompiler().compile(sequential.sequential_pipeline, package_path)
      with open(os.path.join(test_data_dir, 'sequential.yaml'), 'r') as f:
        golden = yaml.safe_load(f)
      compiled = self._get_yaml_from_zip(package_path)
      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)
