# Copyright 2020 Google LLC
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

# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest
import datetime

import kfp_server_api
from kfp_server_api.models.api_pipeline_version import ApiPipelineVersion  # noqa: E501
from kfp_server_api.rest import ApiException

class TestApiPipelineVersion(unittest.TestCase):
    """ApiPipelineVersion unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test ApiPipelineVersion
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_server_api.models.api_pipeline_version.ApiPipelineVersion()  # noqa: E501
        if include_optional :
            return ApiPipelineVersion(
                id = '0', 
                name = '0', 
                created_at = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                parameters = [
                    kfp_server_api.models.api_parameter.apiParameter(
                        name = '0', 
                        value = '0', )
                    ], 
                code_source_url = '0', 
                package_url = kfp_server_api.models.api_url.apiUrl(
                    pipeline_url = '0', ), 
                resource_references = [
                    kfp_server_api.models.api_resource_reference.apiResourceReference(
                        key = kfp_server_api.models.api_resource_key.apiResourceKey(
                            type = 'UNKNOWN_RESOURCE_TYPE', 
                            id = '0', ), 
                        name = '0', 
                        relationship = 'UNKNOWN_RELATIONSHIP', )
                    ]
            )
        else :
            return ApiPipelineVersion(
        )

    def testApiPipelineVersion(self):
        """Test ApiPipelineVersion"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
