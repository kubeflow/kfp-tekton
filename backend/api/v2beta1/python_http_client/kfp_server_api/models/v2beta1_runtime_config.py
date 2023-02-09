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

from kfp_server_api.configuration import Configuration


class V2beta1RuntimeConfig(object):
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
        'parameters': 'dict(str, ProtobufValue)',
        'pipeline_root': 'str'
    }

    attribute_map = {
        'parameters': 'parameters',
        'pipeline_root': 'pipeline_root'
    }

    def __init__(self, parameters=None, pipeline_root=None, local_vars_configuration=None):  # noqa: E501
        """V2beta1RuntimeConfig - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._parameters = None
        self._pipeline_root = None
        self.discriminator = None

        if parameters is not None:
            self.parameters = parameters
        if pipeline_root is not None:
            self.pipeline_root = pipeline_root

    @property
    def parameters(self):
        """Gets the parameters of this V2beta1RuntimeConfig.  # noqa: E501

        The runtime parameters of the Pipeline. The parameters will be used to replace the placeholders at runtime.  # noqa: E501

        :return: The parameters of this V2beta1RuntimeConfig.  # noqa: E501
        :rtype: dict(str, ProtobufValue)
        """
        return self._parameters

    @parameters.setter
    def parameters(self, parameters):
        """Sets the parameters of this V2beta1RuntimeConfig.

        The runtime parameters of the Pipeline. The parameters will be used to replace the placeholders at runtime.  # noqa: E501

        :param parameters: The parameters of this V2beta1RuntimeConfig.  # noqa: E501
        :type parameters: dict(str, ProtobufValue)
        """

        self._parameters = parameters

    @property
    def pipeline_root(self):
        """Gets the pipeline_root of this V2beta1RuntimeConfig.  # noqa: E501


        :return: The pipeline_root of this V2beta1RuntimeConfig.  # noqa: E501
        :rtype: str
        """
        return self._pipeline_root

    @pipeline_root.setter
    def pipeline_root(self, pipeline_root):
        """Sets the pipeline_root of this V2beta1RuntimeConfig.


        :param pipeline_root: The pipeline_root of this V2beta1RuntimeConfig.  # noqa: E501
        :type pipeline_root: str
        """

        self._pipeline_root = pipeline_root

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
        if not isinstance(other, V2beta1RuntimeConfig):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V2beta1RuntimeConfig):
            return True

        return self.to_dict() != other.to_dict()
