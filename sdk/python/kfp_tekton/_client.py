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
import tempfile

from datetime import datetime
from typing import Mapping, Callable

import kfp

from .compiler import TektonCompiler


class TektonClient(kfp.Client):
  """Tekton API Client for Kubeflow Pipelines."""

  def create_run_from_pipeline_func(self,
                                    pipeline_func: Callable,
                                    arguments: Mapping[str, str],
                                    run_name=None,
                                    experiment_name=None,
                                    pipeline_conf: kfp.dsl.PipelineConf = None,
                                    namespace=None):
    """Runs pipeline on Kubernetes cluster with Kubeflow Pipelines Tekton backend.

    This command compiles the pipeline function, creates or gets an experiment and
    submits the pipeline for execution.

    :param pipeline_func: A function that describes a pipeline by calling components
        and composing them into execution graph.
    :param arguments: Arguments to the pipeline function provided as a dict.
    :param run_name: Optional. Name of the run to be shown in the UI.
    :param experiment_name: Optional. Name of the experiment to add the run to.
    :param pipeline_conf: Optional. Pipeline configuration.
    :param namespace: kubernetes namespace where the pipeline runs are created.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized
    :return: RunPipelineResult
    """

    # TODO: Check arguments against the pipeline function
    pipeline_name = pipeline_func.__name__
    run_name = run_name or pipeline_name + ' ' + datetime.now().strftime('%Y-%m-%d %H-%M-%S')
    try:
      (_, pipeline_package_path) = tempfile.mkstemp(suffix='.zip')
      TektonCompiler().compile(pipeline_func, pipeline_package_path, pipeline_conf=pipeline_conf)
      return self.create_run_from_pipeline_package(pipeline_package_path, arguments,
                                                   run_name, experiment_name, namespace)
    finally:
      os.remove(pipeline_package_path)
