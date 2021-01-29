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
from kfp.components import func_to_container_op
from kfp import onprem
from kfp_tekton.k8s_client_helper import env_from_secret


@func_to_container_op
def write_file(output_text_path: str):
    with open(output_text_path, 'w') as writer:
        writer.write('hello world')


email_op = kfp.components.load_component_from_file('../../components/notification/component.yaml')
# pvc mount point has to be string, not pipeline param.
attachment_path = "/tmp/data"


@dsl.pipeline(
    name='email_pipeline',
    description='email pipeline'
)
def email_pipeline(
    server_secret="server-secret",
    subject="Hi, again!",
    body="Tekton email",
    sender="me@myserver.com",
    recipients="him@hisserver.com, her@herserver.com",
    attachment_filepath="/tmp/data/output.txt"
):
    email = email_op(server_secret=server_secret,
                     subject=subject,
                     body=body,
                     sender=sender,
                     recipients=recipients,
                     attachment_path=attachment_filepath)
    email.add_env_variable(env_from_secret('USER', '$(params.server_secret)', 'user'))
    email.add_env_variable(env_from_secret('PASSWORD', '$(params.server_secret)', 'password'))
    email.add_env_variable(env_from_secret('TLS', '$(params.server_secret)', 'tls'))
    email.add_env_variable(env_from_secret('SERVER', '$(params.server_secret)', 'url'))
    email.add_env_variable(env_from_secret('PORT', '$(params.server_secret)', 'port'))
    email.apply(onprem.mount_pvc('shared-pvc', 'shared-pvc', attachment_path))

    with dsl.ExitHandler(email):
        write_file_task = write_file(attachment_filepath).apply(onprem.mount_pvc('shared-pvc', 'shared-pvc', attachment_path))


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(email_pipeline, 'email_pipeline.yaml')
