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
from kfp_tekton_server_api.models.v1_get_healthz_response import V1GetHealthzResponse  # noqa: E501
from kfp_tekton_server_api.rest import ApiException

class TestV1GetHealthzResponse(unittest.TestCase):
    """V1GetHealthzResponse unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1GetHealthzResponse
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_tekton_server_api.models.v1_get_healthz_response.V1GetHealthzResponse()  # noqa: E501
        if include_optional :
            return V1GetHealthzResponse(
                multi_user = True
            )
        else :
            return V1GetHealthzResponse(
        )

    def testV1GetHealthzResponse(self):
        """Test V1GetHealthzResponse"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
