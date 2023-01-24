# Copyright 2019-2020 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from kubernetes import client as k8s_client


def env_from_secret(env_name, secret_name, secret_key):
    """ Creates a V1EnvVar object from a secret.

        Parameters
        ----------
        env_name : str
            The name of the environment variable.
        secret_name : str
            The name of the secret.
        secret_key : str
            The key of the secret.

        Returns
        -------
        V1EnvVar
            A V1EnvVar object.
    """
    return k8s_client.V1EnvVar(
        name=env_name,
        value_from=k8s_client.V1EnvVarSource(
            secret_key_ref=k8s_client.V1SecretKeySelector(
                name=secret_name,
                key=secret_key
            )
        )
    )
