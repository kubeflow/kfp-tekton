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

import kfp_server_api_v1
from kfp_server_api_v1.models.v1_cron_schedule import V1CronSchedule  # noqa: E501
from kfp_server_api_v1.rest import ApiException

class TestV1CronSchedule(unittest.TestCase):
    """V1CronSchedule unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1CronSchedule
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_server_api_v1.models.v1_cron_schedule.V1CronSchedule()  # noqa: E501
        if include_optional :
            return V1CronSchedule(
                start_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                end_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                cron = '0'
            )
        else :
            return V1CronSchedule(
        )

    def testV1CronSchedule(self):
        """Test V1CronSchedule"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
