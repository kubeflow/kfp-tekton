# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from kfp_tekton_server_api.configuration import Configuration


class V1CronSchedule(object):
    """NOTE: This class is auto generated by OpenAPI Generator.
    Ref: https://openapi-generator.tech

    Do not edit the class manually.
    """

    """
    Attributes:
      openapi_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    openapi_types = {
        'start_time': 'datetime',
        'end_time': 'datetime',
        'cron': 'str'
    }

    attribute_map = {
        'start_time': 'startTime',
        'end_time': 'endTime',
        'cron': 'cron'
    }

    def __init__(self, start_time=None, end_time=None, cron=None, local_vars_configuration=None):  # noqa: E501
        """V1CronSchedule - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._start_time = None
        self._end_time = None
        self._cron = None
        self.discriminator = None

        if start_time is not None:
            self.start_time = start_time
        if end_time is not None:
            self.end_time = end_time
        if cron is not None:
            self.cron = cron

    @property
    def start_time(self):
        """Gets the start_time of this V1CronSchedule.  # noqa: E501


        :return: The start_time of this V1CronSchedule.  # noqa: E501
        :rtype: datetime
        """
        return self._start_time

    @start_time.setter
    def start_time(self, start_time):
        """Sets the start_time of this V1CronSchedule.


        :param start_time: The start_time of this V1CronSchedule.  # noqa: E501
        :type start_time: datetime
        """

        self._start_time = start_time

    @property
    def end_time(self):
        """Gets the end_time of this V1CronSchedule.  # noqa: E501


        :return: The end_time of this V1CronSchedule.  # noqa: E501
        :rtype: datetime
        """
        return self._end_time

    @end_time.setter
    def end_time(self, end_time):
        """Sets the end_time of this V1CronSchedule.


        :param end_time: The end_time of this V1CronSchedule.  # noqa: E501
        :type end_time: datetime
        """

        self._end_time = end_time

    @property
    def cron(self):
        """Gets the cron of this V1CronSchedule.  # noqa: E501


        :return: The cron of this V1CronSchedule.  # noqa: E501
        :rtype: str
        """
        return self._cron

    @cron.setter
    def cron(self, cron):
        """Sets the cron of this V1CronSchedule.


        :param cron: The cron of this V1CronSchedule.  # noqa: E501
        :type cron: str
        """

        self._cron = cron

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.openapi_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(map(
                    lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
                    value
                ))
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(map(
                    lambda item: (item[0], item[1].to_dict())
                    if hasattr(item[1], "to_dict") else item,
                    value.items()
                ))
            else:
                result[attr] = value

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, V1CronSchedule):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1CronSchedule):
            return True

        return self.to_dict() != other.to_dict()
