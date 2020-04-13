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
import importlib
import kfp_tekton.compiler as compiler
import filecmp

def _get_yaml_from_zip(zip_file):
    with zipfile.ZipFile(zip_file, 'r') as zip:
        with open(zip.extract(zip.namelist()[0]), 'r') as yaml_file:
            return list(yaml.safe_load_all(yaml_file))

def get_config(config_path):
    with open(config_path) as file:
        return list(yaml.safe_load_all(file))

def get_params_from_config(pipeline_name, config_path):
    pipelines = get_config(config_path)

    for pipeline in pipelines:
        if pipeline_name == pipeline["pipeline"]:
            return pipeline

def test_workflow_without_decorator(pipeline_mod, params_dict):
    """Test compiling a workflow and appending pipeline params."""

    try:
        pipeline_params = []
        for param in params_dict.get('paramsList', []):
            pipeline_params.append(getattr(pipeline_mod, param))

        compiled_workflow = compiler.TektonCompiler()._create_workflow(
            getattr(pipeline_mod,params_dict['function']),
            params_dict.get('name', None),
            params_dict.get('description', None),
            pipeline_params if pipeline_params else None,
            params_dict.get('conf', None))
        return True
    except :
        return False

def test_nested_workflow(pipeline_mod, pipeline_list):
    """Test compiling a simple workflow, and a bigger one composed from the simple one."""

    tmpdir = tempfile.mkdtemp()
    try:
        for pipeline in pipeline_list:
            pipeline_name = pipeline['name']
            package_path = os.path.join(tmpdir, pipeline_name + '.zip')
            compiler.TektonCompiler().compile(getattr(pipeline_mod, pipeline_name), package_path)
        return True
    except:
        return False


if __name__ == '__main__':
    test_data_path = sys.argv[1]
    config_path = sys.argv[2]
    did_compile = False

    # Import pipeline
    test_data_dir, test_data_file = os.path.split(test_data_path)
    import_name, test_data_ext = os.path.splitext(test_data_file)
    sys.path.append(test_data_dir)
    pipeline_mod = importlib.import_module(import_name)

    # Get the pipeline specific parameters from the config file
    params = get_params_from_config(test_data_file, config_path)
    test_type = params['type']

    if test_type == 'nested':
        did_compile = test_nested_workflow(pipeline_mod, params['components'])
    elif test_type == 'no_decorator':
        did_compile = test_workflow_without_decorator(pipeline_mod, params['components'])
    else:
        raise ValueError('No pipeline matches available')

    if did_compile:
        print("SUCCESS: ", test_data_file)
    else:
        print("FAILURE: ", test_data_file)
