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
import time

from datetime import datetime
from typing import Mapping, Callable

from kfp_server_api import ApiException

import kfp
import logging

from .compiler import TektonCompiler
from .compiler.pipeline_utils import TektonPipelineConf


class TektonClient(kfp.Client):
  """Tekton API Client for Kubeflow Pipelines."""

  def create_run_from_pipeline_func(self,
                                    pipeline_func: Callable,
                                    arguments: Mapping[str, str],
                                    run_name=None,
                                    experiment_name=None,
                                    pipeline_conf: kfp.dsl.PipelineConf = None,
                                    tekton_pipeline_conf: TektonPipelineConf = None,
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
      TektonCompiler().compile(pipeline_func,
                               pipeline_package_path,
                               pipeline_conf=pipeline_conf,
                               tekton_pipeline_conf=tekton_pipeline_conf)
      return self.create_run_from_pipeline_package(pipeline_package_path, arguments,
                                                   run_name, experiment_name, namespace)
    finally:
      os.remove(pipeline_package_path)

  def wait_for_run_completion(self, run_id: str, timeout: int):
    """Waits for a run to complete.

    Args:
      run_id: Run id, returned from run_pipeline.
      timeout: Timeout in seconds.

    Returns:
      A run detail object: Most important fields are run and pipeline_runtime.

    Raises:
      TimeoutError: if the pipeline run failed to finish before the specified timeout.
    """
    status = 'Running:'
    start_time = datetime.datetime.now()
    if isinstance(timeout, datetime.timedelta):
      timeout = timeout.total_seconds()
    is_valid_token = False
    # Tekton pipelineruns Status
    # List of Tekton status: https://github.com/tektoncd/pipeline/blob/main/pkg/apis/pipeline/v1/pipelinerun_types.go
    # List of Tekton error status: https://github.com/tektoncd/pipeline/blob/main/pkg/reconciler/pipelinerun/pipelinerun.go
    tekton_completion_status = {'succeeded',
                                'failed',
                                'skipped',
                                'error',
                                'completed',
                                'pipelineruncancelled',
                                'pipelineruncouldntcancel',
                                'pipelineruntimeout',
                                'cancelled',
                                'stoppedrunfinally',
                                'cancelledrunfinally',
                                'invalidtaskresultreference'}
    while True:
      try:
        get_run_response = self._run_api.get_run(run_id=run_id)
        is_valid_token = True
      except ApiException as api_ex:
        # if the token is valid but receiving 401 Unauthorized error
        # then refresh the token
        if is_valid_token and api_ex.status == 401:
          logging.info('Access token has expired !!! Refreshing ...')
          self._refresh_api_client_token()
          continue
        else:
          raise api_ex
      status = get_run_response.run.status
      elapsed_time = (datetime.datetime.now() -
                      start_time).total_seconds()
      logging.info('Waiting for the job to complete...')
      if elapsed_time > timeout:
        raise TimeoutError('Run timeout')
      if status is not None and status.lower() in tekton_completion_status:
        return get_run_response
      time.sleep(5)
