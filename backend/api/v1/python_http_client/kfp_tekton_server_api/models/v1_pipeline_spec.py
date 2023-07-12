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


class V1PipelineSpec(object):
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
        'pipeline_id': 'str',
        'pipeline_name': 'str',
        'workflow_manifest': 'str',
        'pipeline_manifest': 'str',
        'parameters': 'list[V1Parameter]',
        'runtime_config': 'PipelineSpecRuntimeConfig'
    }

    attribute_map = {
        'pipeline_id': 'pipelineId',
        'pipeline_name': 'pipelineName',
        'workflow_manifest': 'workflowManifest',
        'pipeline_manifest': 'pipelineManifest',
        'parameters': 'parameters',
        'runtime_config': 'runtimeConfig'
    }

    def __init__(self, pipeline_id=None, pipeline_name=None, workflow_manifest=None, pipeline_manifest=None, parameters=None, runtime_config=None, local_vars_configuration=None):  # noqa: E501
        """V1PipelineSpec - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._pipeline_id = None
        self._pipeline_name = None
        self._workflow_manifest = None
        self._pipeline_manifest = None
        self._parameters = None
        self._runtime_config = None
        self.discriminator = None

        if pipeline_id is not None:
            self.pipeline_id = pipeline_id
        if pipeline_name is not None:
            self.pipeline_name = pipeline_name
        if workflow_manifest is not None:
            self.workflow_manifest = workflow_manifest
        if pipeline_manifest is not None:
            self.pipeline_manifest = pipeline_manifest
        if parameters is not None:
            self.parameters = parameters
        if runtime_config is not None:
            self.runtime_config = runtime_config

    @property
    def pipeline_id(self):
        """Gets the pipeline_id of this V1PipelineSpec.  # noqa: E501

        Optional input field. The ID of the pipeline user uploaded before.  # noqa: E501

        :return: The pipeline_id of this V1PipelineSpec.  # noqa: E501
        :rtype: str
        """
        return self._pipeline_id

    @pipeline_id.setter
    def pipeline_id(self, pipeline_id):
        """Sets the pipeline_id of this V1PipelineSpec.

        Optional input field. The ID of the pipeline user uploaded before.  # noqa: E501

        :param pipeline_id: The pipeline_id of this V1PipelineSpec.  # noqa: E501
        :type pipeline_id: str
        """

        self._pipeline_id = pipeline_id

    @property
    def pipeline_name(self):
        """Gets the pipeline_name of this V1PipelineSpec.  # noqa: E501

        Optional output field. The name of the pipeline. Not empty if the pipeline id is not empty.  # noqa: E501

        :return: The pipeline_name of this V1PipelineSpec.  # noqa: E501
        :rtype: str
        """
        return self._pipeline_name

    @pipeline_name.setter
    def pipeline_name(self, pipeline_name):
        """Sets the pipeline_name of this V1PipelineSpec.

        Optional output field. The name of the pipeline. Not empty if the pipeline id is not empty.  # noqa: E501

        :param pipeline_name: The pipeline_name of this V1PipelineSpec.  # noqa: E501
        :type pipeline_name: str
        """

        self._pipeline_name = pipeline_name

    @property
    def workflow_manifest(self):
        """Gets the workflow_manifest of this V1PipelineSpec.  # noqa: E501

        Optional input field. The marshalled raw argo JSON workflow. This will be deprecated when pipeline_manifest is in use.  # noqa: E501

        :return: The workflow_manifest of this V1PipelineSpec.  # noqa: E501
        :rtype: str
        """
        return self._workflow_manifest

    @workflow_manifest.setter
    def workflow_manifest(self, workflow_manifest):
        """Sets the workflow_manifest of this V1PipelineSpec.

        Optional input field. The marshalled raw argo JSON workflow. This will be deprecated when pipeline_manifest is in use.  # noqa: E501

        :param workflow_manifest: The workflow_manifest of this V1PipelineSpec.  # noqa: E501
        :type workflow_manifest: str
        """

        self._workflow_manifest = workflow_manifest

    @property
    def pipeline_manifest(self):
        """Gets the pipeline_manifest of this V1PipelineSpec.  # noqa: E501

        Optional input field. The raw pipeline JSON spec.  # noqa: E501

        :return: The pipeline_manifest of this V1PipelineSpec.  # noqa: E501
        :rtype: str
        """
        return self._pipeline_manifest

    @pipeline_manifest.setter
    def pipeline_manifest(self, pipeline_manifest):
        """Sets the pipeline_manifest of this V1PipelineSpec.

        Optional input field. The raw pipeline JSON spec.  # noqa: E501

        :param pipeline_manifest: The pipeline_manifest of this V1PipelineSpec.  # noqa: E501
        :type pipeline_manifest: str
        """

        self._pipeline_manifest = pipeline_manifest

    @property
    def parameters(self):
        """Gets the parameters of this V1PipelineSpec.  # noqa: E501


        :return: The parameters of this V1PipelineSpec.  # noqa: E501
        :rtype: list[V1Parameter]
        """
        return self._parameters

    @parameters.setter
    def parameters(self, parameters):
        """Sets the parameters of this V1PipelineSpec.


        :param parameters: The parameters of this V1PipelineSpec.  # noqa: E501
        :type parameters: list[V1Parameter]
        """

        self._parameters = parameters

    @property
    def runtime_config(self):
        """Gets the runtime_config of this V1PipelineSpec.  # noqa: E501


        :return: The runtime_config of this V1PipelineSpec.  # noqa: E501
        :rtype: PipelineSpecRuntimeConfig
        """
        return self._runtime_config

    @runtime_config.setter
    def runtime_config(self, runtime_config):
        """Sets the runtime_config of this V1PipelineSpec.


        :param runtime_config: The runtime_config of this V1PipelineSpec.  # noqa: E501
        :type runtime_config: PipelineSpecRuntimeConfig
        """

        self._runtime_config = runtime_config

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
        if not isinstance(other, V1PipelineSpec):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1PipelineSpec):
            return True

        return self.to_dict() != other.to_dict()
