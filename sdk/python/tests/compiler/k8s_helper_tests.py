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

from kfp_tekton.compiler._k8s_helper import convert_k8s_obj_to_json, \
    sanitize_k8s_name


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
            list(map(lambda k: sanitize_k8s_name(k, True, True, True, 253), labels.keys())),
            list(expected_labels.keys()))
        self.assertEqual(
            list(map(lambda v: sanitize_k8s_name(v, True, True, False, 63), labels.values())),
            list(expected_labels.values()))

    def test_sanitize_k8s_annotations(self):
        annotation_keys = {
            "sidecar.istio.io/inject",
        }
        expected_k8s_annotation_keys = {
            "sidecar.istio.io/inject",
        }
        self.assertEqual(
            [sanitize_k8s_name(key, True, True, True, 253) for key in annotation_keys],
            [key[:253] for key in expected_k8s_annotation_keys])
