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

import kfp_tekton_server_api
from kfp_tekton_server_api.models.v1_pipeline_spec import V1PipelineSpec  # noqa: E501
from kfp_tekton_server_api.rest import ApiException

class TestV1PipelineSpec(unittest.TestCase):
    """V1PipelineSpec unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1PipelineSpec
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_tekton_server_api.models.v1_pipeline_spec.V1PipelineSpec()  # noqa: E501
        if include_optional :
            return V1PipelineSpec(
                pipeline_id = '0', 
                pipeline_name = '0', 
                workflow_manifest = '0', 
                pipeline_manifest = '0', 
                parameters = [
                    kfp_tekton_server_api.models.v1_parameter.v1Parameter(
                        name = '0', 
                        value = '0', )
                    ], 
                runtime_config = kfp_tekton_server_api.models.pipeline_spec_runtime_config.PipelineSpecRuntimeConfig(
                    parameters = {
                        'key' : kfp_tekton_server_api.models.v1_value.v1Value(
                            int_value = '0', 
                            double_value = 1.337, 
                            string_value = '0', )
                        }, 
                    pipeline_root = '0', )
            )
        else :
            return V1PipelineSpec(
        )

    def testV1PipelineSpec(self):
        """Test V1PipelineSpec"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
