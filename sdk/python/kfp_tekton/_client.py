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
import warnings
import kfp_server_api
from kfp._auth import get_auth_token, get_gcp_access_token

from datetime import datetime
from typing import Mapping, Callable

from kfp_server_api import ApiException

import kfp
import logging

from .compiler import TektonCompiler
from .compiler.pipeline_utils import TektonPipelineConf

KF_PIPELINES_ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
KF_PIPELINES_UI_ENDPOINT_ENV = 'KF_PIPELINES_UI_ENDPOINT'
KF_PIPELINES_DEFAULT_EXPERIMENT_NAME = 'KF_PIPELINES_DEFAULT_EXPERIMENT_NAME'
KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME = 'KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'
KF_PIPELINES_IAP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_IAP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET'


class TektonClient(kfp.Client):
    """Tekton API Client for Kubeflow Pipelines."""

    def __init__(self,
                 host=None,
                 client_id=None,
                 namespace='kubeflow',
                 other_client_id=None,
                 other_client_secret=None,
                 existing_token=None,
                 cookies=None,
                 proxy=None,
                 ssl_ca_cert=None,
                 kube_context=None,
                 credentials=None,
                 ui_host=None,
                 verify_ssl=None):
        """Create a new instance of kfp client."""
        host = host or os.environ.get(KF_PIPELINES_ENDPOINT_ENV)
        self._uihost = os.environ.get(KF_PIPELINES_UI_ENDPOINT_ENV, ui_host or
                                      host)
        client_id = client_id or os.environ.get(
            KF_PIPELINES_IAP_OAUTH2_CLIENT_ID_ENV)
        other_client_id = other_client_id or os.environ.get(
            KF_PIPELINES_APP_OAUTH2_CLIENT_ID_ENV)
        other_client_secret = other_client_secret or os.environ.get(
            KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET_ENV)
        config = self._load_config(host, client_id, namespace, other_client_id,
                                   other_client_secret, existing_token, proxy,
                                   ssl_ca_cert, kube_context, credentials,
                                   verify_ssl)
        # Save the loaded API client configuration, as a reference if update is
        # needed.
        self._load_context_setting_or_default()

        # If custom namespace provided, overwrite the loaded or default one in
        # context settings for current client instance
        if namespace != 'kubeflow':
            self._context_setting['namespace'] = namespace

        self._existing_config = config
        if cookies is None:
            cookies = self._context_setting.get('client_authentication_cookie')
        api_client = kfp_server_api.api_client.ApiClient(
            config,
            cookie=cookies,
            header_name=self._context_setting.get(
                'client_authentication_header_name'),
            header_value=self._context_setting.get(
                'client_authentication_header_value'))
        _add_generated_apis(self, kfp_server_api, api_client)
        self._job_api = kfp_server_api.api.job_service_api.JobServiceApi(
            api_client)
        self._run_api = kfp_server_api.api.run_service_api.RunServiceApi(
            api_client)
        self._experiment_api = kfp_server_api.api.experiment_service_api.ExperimentServiceApi(
            api_client)
        self._pipelines_api = kfp_server_api.api.pipeline_service_api.PipelineServiceApi(
            api_client)
        self._upload_api = kfp_server_api.api.PipelineUploadServiceApi(
            api_client)
        self._healthz_api = kfp_server_api.api.healthz_service_api.HealthzServiceApi(
            api_client)
        if not self._context_setting['namespace'] and self.get_kfp_healthz(
        ).multi_user is True:
            try:
                with open(TektonClient.NAMESPACE_PATH, 'r') as f:
                    current_namespace = f.read()
                    self.set_user_namespace(current_namespace)
            except FileNotFoundError:
                logging.info(
                    'Failed to automatically set namespace.', exc_info=False)

    def _load_config(self, host, client_id, namespace, other_client_id,
                     other_client_secret, existing_token, proxy, ssl_ca_cert,
                     kube_context, credentials, verify_ssl):
        config = kfp_server_api.configuration.Configuration()

        if proxy:
            # https://github.com/kubeflow/pipelines/blob/c6ac5e0b1fd991e19e96419f0f508ec0a4217c29/backend/api/python_http_client/kfp_server_api/rest.py#L100
            config.proxy = proxy
        if verify_ssl is not None:
            config.verify_ssl = verify_ssl

        if ssl_ca_cert:
            config.ssl_ca_cert = ssl_ca_cert

        host = host or ''

        # Defaults to 'https' if host does not contain 'http' or 'https' protocol.
        if host and not host.startswith('http'):
            warnings.warn(
                'The host %s does not contain the "http" or "https" protocol.'
                ' Defaults to "https".' % host)
            host = 'https://' + host

        # Preprocess the host endpoint to prevent some common user mistakes.
        if not client_id:
            # always preserving the protocol (http://localhost requires it)
            host = host.rstrip('/')

        if host:
            config.host = host

        token = None

        # "existing_token" is designed to accept token generated outside of SDK. Here is an example.
        #
        # https://cloud.google.com/functions/docs/securing/function-identity
        # https://cloud.google.com/endpoints/docs/grpc/service-account-authentication
        #
        # import requests
        # import kfp
        #
        # def get_access_token():
        #     url = 'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token'
        #     r = requests.get(url, headers={'Metadata-Flavor': 'Google'})
        #     r.raise_for_status()
        #     access_token = r.json()['access_token']
        #     return access_token
        #
        # client = kfp.Client(host='<KFPHost>', existing_token=get_access_token())
        #
        if existing_token:
            token = existing_token
            self._is_refresh_token = False
        elif client_id:
            token = get_auth_token(client_id, other_client_id,
                                   other_client_secret)
            self._is_refresh_token = True
        elif self._is_inverse_proxy_host(host):
            token = get_gcp_access_token()
            self._is_refresh_token = False
        elif credentials:
            config.api_key['authorization'] = 'placeholder'
            config.api_key_prefix['authorization'] = 'Bearer'
            config.refresh_api_key_hook = credentials.refresh_api_key_hook

        if token:
            config.api_key['authorization'] = token
            config.api_key_prefix['authorization'] = 'Bearer'
            return config

        if host:
            # if host is explicitly set with auth token, it's probably a port forward address.
            return config

        import kubernetes as k8s
        in_cluster = True
        try:
            k8s.config.load_incluster_config()
        except:
            in_cluster = False
            pass

        if in_cluster:
            config.host = TektonClient.IN_CLUSTER_DNS_NAME.format(namespace)
            config = self._get_config_with_default_credentials(config)
            return config

        try:
            k8s.config.load_kube_config(
                client_configuration=config, context=kube_context)
        except:
            print('Failed to load kube config.')
            return config

        if config.host:
            config.host = config.host + '/' + TektonClient.KUBE_PROXY_PATH.format(
                namespace)
        return config

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
      run_name = run_name or pipeline_name + ' ' + datetime.datetime.now().strftime('%Y-%m-%d %H-%M-%S')
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
                get_run_response = self._run_api.run_service_get_run(run_id=run_id)
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

def _add_generated_apis(target_struct, api_module, api_client):
    """Initializes a hierarchical API object based on the generated API module.

    PipelineServiceApi.create_pipeline becomes
    target_struct.pipelines.create_pipeline
    """
    Struct = type('Struct', (), {})

    def camel_case_to_snake_case(name):
        import re
        return re.sub('([a-z0-9])([A-Z])', r'\1_\2', name).lower()

    for api_name in dir(api_module):
        if not api_name.endswith('ServiceApi'):
            continue

        short_api_name = camel_case_to_snake_case(
            api_name[0:-len('ServiceApi')]) + 's'
        api_struct = Struct()
        setattr(target_struct, short_api_name, api_struct)
        service_api = getattr(api_module.api, api_name)
        initialized_service_api = service_api(api_client)
        for member_name in dir(initialized_service_api):
            if member_name.startswith('_') or member_name.endswith(
                    '_with_http_info'):
                continue

            bound_member = getattr(initialized_service_api, member_name)
            setattr(api_struct, member_name, bound_member)
    models_struct = Struct()
    for member_name in dir(api_module.models):
        if not member_name[0].islower():
            setattr(models_struct, member_name,
                    getattr(api_module.models, member_name))
    target_struct.api_models = models_struct