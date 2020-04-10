#!/bin/bash

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
import sys
import shutil
import zipfile
import yaml
import tempfile
import kfp_tekton.compiler as compiler
import filecmp

def _get_yaml_from_zip(zip_file):
    with zipfile.ZipFile(zip_file, 'r') as zip:
        with open(zip.extract(zip.namelist()[0]), 'r') as yaml_file:
            return list(yaml.safe_load_all(yaml_file))


def test_basic_workflow_without_decorator(test_data_dir):
    """Test compiling a workflow and appending pipeline params."""

    sys.path.append(test_data_dir)
    import basic_no_decorator
    try:
        compiled_workflow = compiler.TektonCompiler()._create_workflow(
            basic_no_decorator.save_most_frequent_word,
            'Save Most Frequent',
            'Get Most Frequent Word and Save to GCS',
            [
                basic_no_decorator.message_param,
                basic_no_decorator.output_path_param
            ])
        print("SUCCESS: basic_no_decorator.py")
    except:
        print("FAILURE: basic_no_decorator.py")

def test_composing_workflow(test_data_dir):
    """Test compiling a simple workflow, and a bigger one composed from the simple one."""

    sys.path.append(test_data_dir)
    import compose
    tmpdir = tempfile.mkdtemp()
    try:
        # First make sure the simple pipeline can be compiled.
        simple_package_path = os.path.join(tmpdir, 'simple.zip')
        compiler.TektonCompiler().compile(compose.save_most_frequent_word, simple_package_path)

        # Then make sure the composed pipeline can be compiled and also compare with golden.
        compose_package_path = os.path.join(tmpdir, 'compose.zip')
        compiler.TektonCompiler().compile(compose.download_save_most_frequent_word, compose_package_path)

        print("SUCCESS: compose.py")
    except:
        print("FAILURE: compose.py")


if __name__ == '__main__':
    test_name = sys.argv[1]
    test_data_dir = sys.argv[2]
    if test_name == 'compose.py':
        test_composing_workflow(test_data_dir)
    elif test_name == 'basic_no_decorator.py':
        test_basic_workflow_without_decorator(test_data_dir)
    else:
        raise ValueError('No pipeline matches available')
