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

from kfp import dsl
import kfp
from kubernetes import client as k8s_client


def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Got scheduled"']
    )


def env_from_secret(env_name, secret_name, secret_key):
    return k8s_client.V1EnvVar(
        name=env_name,
        value_from=k8s_client.V1EnvVarSource(
            secret_key_ref=k8s_client.V1SecretKeySelector(
                name=secret_name,
                key=secret_key
            )
        )
    )


email_op = kfp.components.load_component_from_file('component.yaml')


@dsl.pipeline(
    name='email_pipeline',
    description='email pipeline'
)
def email_pipeline(
    server="server-secret",
    subject="Hi, again!",
    body="Tekton email",
    sender="<me@myserver.com>",
    recipients="<him@hisserver.com> <her@herserver.com>"
):
    email = email_op(server=server,
                     subject=subject,
                     body=body,
                     sender=sender,
                     recipients=recipients)
    email.add_env_variable(env_from_secret('USER', '$(params.server)', 'user'))
    email.add_env_variable(env_from_secret('PASSWORD', '$(params.server)', 'password'))
    email.add_env_variable(env_from_secret('TLS', '$(params.server)', 'tls'))
    email.add_env_variable(env_from_secret('SERVER', '$(params.server)', 'url'))
    email.add_env_variable(env_from_secret('PORT', '$(params.server)', 'port'))

    with dsl.ExitHandler(email):
        echo = echo_op()



if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(email_pipeline, 'email_pipeline.yaml')
