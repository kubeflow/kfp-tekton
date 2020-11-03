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

import unittest

from datetime import datetime

from kfp import dsl
from kubernetes import client as k8s_client
from kfp_tekton.compiler._k8s_helper import convert_k8s_obj_to_json, \
    sanitize_k8s_name, sanitize_k8s_object


class TestK8sHelper(unittest.TestCase):

    def test_convert_k8s_obj_to_json_accepts_dict(self):
        now = datetime.now()
        converted = convert_k8s_obj_to_json({
            "ENV": "test",
            "number": 3,
            "list": [1, 2, 3],
            "time": now
        })
        self.assertEqual(converted, {
            "ENV": "test",
            "number": 3,
            "list": [1, 2, 3],
            "time": now.isoformat()
        })

    def test_sanitize_k8s_name_max_length(self):
        from string import ascii_lowercase, ascii_uppercase, digits, punctuation
        names = [
            "short-name with under_score and spaces",
            "very long name".replace("o", "o" * 300),
            digits + ascii_uppercase + punctuation + digits
        ]
        expected_names = [
            "short-name-with-under-score-and-spaces",
            "very-long-name".replace("o", "o" * 300),
            digits + ascii_lowercase + "-" + digits
        ]
        self.assertEqual(
            [sanitize_k8s_name(name) for name in names],
            [name[:63] for name in expected_names])
        self.assertEqual(
            [sanitize_k8s_name(sanitize_k8s_name(name)) for name in names],
            [name[:63] for name in expected_names])

    def test_sanitize_k8s_labels(self):
        labels = {
            "my.favorite/hobby": "Hobby? Passion! Football. Go to https://www.fifa.com/",
            "My other hobbies?": "eating; drinking. sleeping ;-)"
        }
        expected_labels = {
            "my.favorite/hobby": "Hobby-Passion-Football.-Go-to-https-www.fifa.com",
            "My-other-hobbies": "eating-drinking.-sleeping"
        }
        self.assertEqual(
            sorted(map(lambda k: sanitize_k8s_name(k, allow_capital_underscore=True, allow_dot=True,
                                                   allow_slash=True, max_length=253), labels.keys())),
            sorted(expected_labels.keys()))
        self.assertEqual(
            sorted(map(lambda v: sanitize_k8s_name(v, allow_capital_underscore=True, allow_dot=True,
                                                   allow_slash=False, max_length=63), labels.values())),
            sorted(expected_labels.values()))

    def test_sanitize_k8s_annotations(self):
        annotation_keys = {
            "sidecar.istio.io/inject",
        }
        expected_k8s_annotation_keys = {
            "sidecar.istio.io/inject",
        }
        self.assertEqual(
            [sanitize_k8s_name(key, allow_capital_underscore=True, allow_dot=True, allow_slash=True,
                               max_length=253) for key in annotation_keys],
            [key[:253] for key in expected_k8s_annotation_keys])

    def test_sanitize_k8s_container_attribute(self):
        # test cases for implicit type sanitization(conversion)
        op = dsl.ContainerOp(name='echo', image='image', command=['sh', '-c'],
                           arguments=['echo test | tee /tmp/message.txt'],
                           file_outputs={'merged': '/tmp/message.txt'})
        op.container \
                .add_volume_mount(k8s_client.V1VolumeMount(
                    mount_path='/secret/gcp-credentials',
                    name='gcp-credentials')) \
                .add_env_variable(k8s_client.V1EnvVar(
                    name=80,
                    value=80)) \
                .add_env_variable(k8s_client.V1EnvVar(
                    name=80,
                    value_from=k8s_client.V1EnvVarSource(
                        config_map_key_ref=k8s_client.V1ConfigMapKeySelector(key=80, name=8080, optional='False'),
                        field_ref=k8s_client.V1ObjectFieldSelector(api_version=80, field_path=8080),
                        resource_field_ref=k8s_client.V1ResourceFieldSelector(container_name=80, divisor=8080, resource=8888),
                        secret_key_ref=k8s_client.V1SecretKeySelector(key=80, name=8080, optional='False')
                    )
                )) \
                .add_env_from(k8s_client.V1EnvFromSource(
                    config_map_ref=k8s_client.V1ConfigMapEnvSource(name=80, optional='True'),
                    prefix=999
                )) \
                .add_env_from(k8s_client.V1EnvFromSource(
                    secret_ref=k8s_client.V1SecretEnvSource(name=80, optional='True'),
                    prefix=888
                )) \
                .add_volume_mount(k8s_client.V1VolumeMount(
                    mount_path=111,
                    mount_propagation=222,
                    name=333,
                    read_only='False',
                    sub_path=444,
                    sub_path_expr=555
                )) \
                .add_volume_devices(k8s_client.V1VolumeDevice(
                    device_path=111,
                    name=222
                )) \
                .add_port(k8s_client.V1ContainerPort(
                    container_port='8080',
                    host_ip=111,
                    host_port='8888',
                    name=222,
                    protocol=333
                )) \
                .set_security_context(k8s_client.V1SecurityContext(
                    allow_privilege_escalation='True',
                    capabilities=k8s_client.V1Capabilities(add=[11, 22], drop=[33, 44]),
                    privileged='False',
                    proc_mount=111,
                    read_only_root_filesystem='False',
                    run_as_group='222',
                    run_as_non_root='True',
                    run_as_user='333',
                    se_linux_options=k8s_client.V1SELinuxOptions(level=11, role=22, type=33, user=44),
                    windows_options=k8s_client.V1WindowsSecurityContextOptions(
                        gmsa_credential_spec=11, gmsa_credential_spec_name=22)
                )) \
                .set_stdin(stdin='False') \
                .set_stdin_once(stdin_once='False') \
                .set_termination_message_path(termination_message_path=111) \
                .set_tty(tty='False') \
                .set_readiness_probe(readiness_probe=k8s_client.V1Probe(
                    _exec=k8s_client.V1ExecAction(command=[11, 22, 33]),
                    failure_threshold='111',
                    http_get=k8s_client.V1HTTPGetAction(
                        host=11,
                        http_headers=[k8s_client.V1HTTPHeader(name=22, value=33)],
                        path=44,
                        port='55',
                        scheme=66),
                    initial_delay_seconds='222',
                    period_seconds='333',
                    success_threshold='444',
                    tcp_socket=k8s_client.V1TCPSocketAction(host=555, port='666'),
                    timeout_seconds='777'
                )) \
                .set_liveness_probe(liveness_probe=k8s_client.V1Probe(
                    _exec=k8s_client.V1ExecAction(command=[11, 22, 33]),
                    failure_threshold='111',
                    http_get=k8s_client.V1HTTPGetAction(
                        host=11,
                        http_headers=[k8s_client.V1HTTPHeader(name=22, value=33)],
                        path=44,
                        port='55',
                        scheme=66),
                    initial_delay_seconds='222',
                    period_seconds='333',
                    success_threshold='444',
                    tcp_socket=k8s_client.V1TCPSocketAction(host=555, port='666'),
                    timeout_seconds='777'
                )) \
                .set_lifecycle(lifecycle=k8s_client.V1Lifecycle(
                    post_start=k8s_client.V1Handler(
                        _exec=k8s_client.V1ExecAction(command=[11, 22, 33]),
                        http_get=k8s_client.V1HTTPGetAction(
                            host=11,
                            http_headers=[k8s_client.V1HTTPHeader(name=22, value=33)],
                            path=44,
                            port='55',
                            scheme=66),
                        tcp_socket=k8s_client.V1TCPSocketAction(host=555, port='666')
                    ),
                    pre_stop=k8s_client.V1Handler(
                        _exec=k8s_client.V1ExecAction(command=[11, 22, 33]),
                        http_get=k8s_client.V1HTTPGetAction(
                            host=11,
                            http_headers=[k8s_client.V1HTTPHeader(name=22, value=33)],
                            path=44,
                            port='55',
                            scheme=66),
                        tcp_socket=k8s_client.V1TCPSocketAction(host=555, port='666')
                    )
                ))

        sanitize_k8s_object(op.container)

        for e in op.container.env:
            self.assertIsInstance(e.name, str)
            if e.value:
                self.assertIsInstance(e.value, str)
            if e.value_from:
                if e.value_from.config_map_key_ref:
                    self.assertIsInstance(e.value_from.config_map_key_ref.key, str)
                    if e.value_from.config_map_key_ref.name:
                        self.assertIsInstance(e.value_from.config_map_key_ref.name, str)
                    if e.value_from.config_map_key_ref.optional:
                        self.assertIsInstance(e.value_from.config_map_key_ref.optional, bool)
                if e.value_from.field_ref:
                    self.assertIsInstance(e.value_from.field_ref.field_path, str)
                    if e.value_from.field_ref.api_version:
                        self.assertIsInstance(e.value_from.field_ref.api_version, str)
                if e.value_from.resource_field_ref:
                    self.assertIsInstance(e.value_from.resource_field_ref.resource, str)
                    if e.value_from.resource_field_ref.container_name:
                        self.assertIsInstance(e.value_from.resource_field_ref.container_name, str)
                    if e.value_from.resource_field_ref.divisor:
                        self.assertIsInstance(e.value_from.resource_field_ref.divisor, str)
                if e.value_from.secret_key_ref:
                    self.assertIsInstance(e.value_from.secret_key_ref.key, str)
                    if e.value_from.secret_key_ref.name:
                        self.assertIsInstance(e.value_from.secret_key_ref.name, str)
                    if e.value_from.secret_key_ref.optional:
                        self.assertIsInstance(e.value_from.secret_key_ref.optional, bool)

        for e in op.container.env_from:
            if e.prefix:
                self.assertIsInstance(e.prefix, str)
            if e.config_map_ref:
                if e.config_map_ref.name:
                    self.assertIsInstance(e.config_map_ref.name, str)
                if e.config_map_ref.optional:
                    self.assertIsInstance(e.config_map_ref.optional, bool)
            if e.secret_ref:
                if e.secret_ref.name:
                    self.assertIsInstance(e.secret_ref.name, str)
                if e.secret_ref.optional:
                    self.assertIsInstance(e.secret_ref.optional, bool)

        for e in op.container.volume_mounts:
            if e.mount_path:
                self.assertIsInstance(e.mount_path, str)
            if e.mount_propagation:
                self.assertIsInstance(e.mount_propagation, str)
            if e.name:
                self.assertIsInstance(e.name, str)
            if e.read_only:
                self.assertIsInstance(e.read_only, bool)
            if e.sub_path:
                self.assertIsInstance(e.sub_path, str)
            if e.sub_path_expr:
                self.assertIsInstance(e.sub_path_expr, str)

        for e in op.container.volume_devices:
            if e.device_path:
                self.assertIsInstance(e.device_path, str)
            if e.name:
                self.assertIsInstance(e.name, str)

        for e in op.container.ports:
            if e.container_port:
                self.assertIsInstance(e.container_port, int)
            if e.host_ip:
                self.assertIsInstance(e.host_ip, str)
            if e.host_port:
                self.assertIsInstance(e.host_port, int)
            if e.name:
                self.assertIsInstance(e.name, str)
            if e.protocol:
                self.assertIsInstance(e.protocol, str)

        if op.container.security_context:
            e = op.container.security_context
            if e.allow_privilege_escalation:
                self.assertIsInstance(e.allow_privilege_escalation, bool)
            if e.capabilities:
                for a in e.capabilities.add:
                    self.assertIsInstance(a, str)
                for d in e.capabilities.drop:
                    self.assertIsInstance(d, str)
            if e.privileged:
                self.assertIsInstance(e.privileged, bool)
            if e.proc_mount:
                self.assertIsInstance(e.proc_mount, str)
            if e.read_only_root_filesystem:
                self.assertIsInstance(e.read_only_root_filesystem, bool)
            if e.run_as_group:
                self.assertIsInstance(e.run_as_group, int)
            if e.run_as_non_root:
                self.assertIsInstance(e.run_as_non_root, bool)
            if e.run_as_user:
                self.assertIsInstance(e.run_as_user, int)
            if e.se_linux_options:
                if e.se_linux_options.level:
                    self.assertIsInstance(e.se_linux_options.level, str)
                if e.se_linux_options.role:
                    self.assertIsInstance(e.se_linux_options.role, str)
                if e.se_linux_options.type:
                    self.assertIsInstance(e.se_linux_options.type, str)
                if e.se_linux_options.user:
                    self.assertIsInstance(e.se_linux_options.user, str)
            if e.windows_options:
                if e.windows_options.gmsa_credential_spec:
                    self.assertIsInstance(e.windows_options.gmsa_credential_spec, str)
                if e.windows_options.gmsa_credential_spec_name:
                    self.assertIsInstance(e.windows_options.gmsa_credential_spec_name, str)
            
        if op.container.stdin:
            self.assertIsInstance(op.container.stdin, bool)

        if op.container.stdin_once:
            self.assertIsInstance(op.container.stdin_once, bool)
        
        if op.container.termination_message_path:
            self.assertIsInstance(op.container.termination_message_path, str)

        if op.container.tty:
            self.assertIsInstance(op.container.tty, bool)

        for e in [op.container.readiness_probe, op.container.liveness_probe]:
            if e:
                if e._exec:
                    for c in e._exec.command:
                        self.assertIsInstance(c, str)
                if e.failure_threshold:
                    self.assertIsInstance(e.failure_threshold, int)
                if e.http_get:
                    if e.http_get.host:
                        self.assertIsInstance(e.http_get.host, str)
                    if e.http_get.http_headers:
                        for h in e.http_get.http_headers:
                            if h.name:
                                self.assertIsInstance(h.name, str)
                            if h.value:
                                self.assertIsInstance(h.value, str)
                    if e.http_get.path:
                        self.assertIsInstance(e.http_get.path, str)
                    if e.http_get.port:
                        self.assertIsInstance(e.http_get.port, (str, int))
                    if e.http_get.scheme:
                        self.assertIsInstance(e.http_get.scheme, str)
                if e.initial_delay_seconds:
                    self.assertIsInstance(e.initial_delay_seconds, int)
                if e.period_seconds:
                    self.assertIsInstance(e.period_seconds, int)
                if e.success_threshold:
                    self.assertIsInstance(e.success_threshold, int)
                if e.tcp_socket:
                    if e.tcp_socket.host:
                        self.assertIsInstance(e.tcp_socket.host, str)
                    if e.tcp_socket.port:
                        self.assertIsInstance(e.tcp_socket.port, (str, int))
                if e.timeout_seconds:
                    self.assertIsInstance(e.timeout_seconds, int)
        if op.container.lifecycle:
            for e in [op.container.lifecycle.post_start, op.container.lifecycle.pre_stop]:
                if e:
                    if e._exec:
                        for c in e._exec.command:
                            self.assertIsInstance(c, str)
                    if e.http_get:
                        if e.http_get.host:
                            self.assertIsInstance(e.http_get.host, str)
                        if e.http_get.http_headers:
                            for h in e.http_get.http_headers:
                                if h.name:
                                    self.assertIsInstance(h.name, str)
                                if h.value:
                                    self.assertIsInstance(h.value, str)
                        if e.http_get.path:
                            self.assertIsInstance(e.http_get.path, str)
                        if e.http_get.port:
                            self.assertIsInstance(e.http_get.port, (str, int))
                        if e.http_get.scheme:
                            self.assertIsInstance(e.http_get.scheme, str)
                    if e.tcp_socket:
                        if e.tcp_socket.host:
                            self.assertIsInstance(e.tcp_socket.host, str)
                        if e.tcp_socket.port:
                            self.assertIsInstance(e.tcp_socket.port, (str, int))

        # test cases for checking value after sanitization
        check_value_op = dsl.ContainerOp(name='echo', image='image', command=['sh', '-c'],
                           arguments=['echo test | tee /tmp/message.txt'],
                           file_outputs={'merged': '/tmp/message.txt'})
        check_value_op.container \
                .add_env_variable(k8s_client.V1EnvVar(
                    name=80,
                    value=8080)) \
                .set_security_context(k8s_client.V1SecurityContext(
                    allow_privilege_escalation='true',
                    capabilities=k8s_client.V1Capabilities(add=[11, 22], drop=[33, 44]),
                    privileged='false',
                    proc_mount=111,
                    read_only_root_filesystem='False',
                    run_as_group='222',
                    run_as_non_root='True',
                    run_as_user='333',
                    se_linux_options=k8s_client.V1SELinuxOptions(level=11, role=22, type=33, user=44),
                    windows_options=k8s_client.V1WindowsSecurityContextOptions(
                        gmsa_credential_spec=11, gmsa_credential_spec_name=22)
                ))
        
        sanitize_k8s_object(check_value_op.container)

        self.assertEqual(check_value_op.container.env[0].name, '80')
        self.assertEqual(check_value_op.container.env[0].value, '8080')
        self.assertEqual(check_value_op.container.security_context.allow_privilege_escalation, True)
        self.assertEqual(check_value_op.container.security_context.capabilities.add[0], '11')
        self.assertEqual(check_value_op.container.security_context.capabilities.add[1], '22')
        self.assertEqual(check_value_op.container.security_context.capabilities.drop[0], '33')
        self.assertEqual(check_value_op.container.security_context.capabilities.drop[1], '44')
        self.assertEqual(check_value_op.container.security_context.privileged, False)
        self.assertEqual(check_value_op.container.security_context.proc_mount, '111')
        self.assertEqual(check_value_op.container.security_context.read_only_root_filesystem, False)
        self.assertEqual(check_value_op.container.security_context.run_as_group, 222)
        self.assertEqual(check_value_op.container.security_context.run_as_non_root, True)
        self.assertEqual(check_value_op.container.security_context.run_as_user, 333)
        self.assertEqual(check_value_op.container.security_context.se_linux_options.level, '11')
        self.assertEqual(check_value_op.container.security_context.se_linux_options.role, '22')
        self.assertEqual(check_value_op.container.security_context.se_linux_options.type, '33')
        self.assertEqual(check_value_op.container.security_context.se_linux_options.user, '44')
        self.assertEqual(check_value_op.container.security_context.windows_options.gmsa_credential_spec, '11')
        self.assertEqual(check_value_op.container.security_context.windows_options.gmsa_credential_spec_name, '22')

        # test cases for exception
        with self.assertRaises(ValueError, msg='Invalid boolean string 2. Should be boolean.'):
            exception_op = dsl.ContainerOp(name='echo', image='image')
            exception_op.container \
                    .set_security_context(k8s_client.V1SecurityContext(
                        allow_privilege_escalation=1
                    ))
            sanitize_k8s_object(exception_op.container)

        with self.assertRaises(ValueError, msg='Invalid boolean string Test. Should be "true" or "false".'):
            exception_op = dsl.ContainerOp(name='echo', image='image')
            exception_op.container \
                    .set_security_context(k8s_client.V1SecurityContext(
                        allow_privilege_escalation='Test'
                    ))
            sanitize_k8s_object(exception_op.container)

        with self.assertRaises(ValueError, msg='Invalid test. Should be integer.'):
            exception_op = dsl.ContainerOp(name='echo', image='image')
            exception_op.container \
                    .set_security_context(k8s_client.V1SecurityContext(
                        run_as_group='test',
                    ))
            sanitize_k8s_object(exception_op.container)
