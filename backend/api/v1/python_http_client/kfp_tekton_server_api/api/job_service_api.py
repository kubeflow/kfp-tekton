# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import re  # noqa: F401

# python 2 and python 3 compatibility library
import six

from kfp_tekton_server_api.api_client import ApiClient
from kfp_tekton_server_api.exceptions import (  # noqa: F401
    ApiTypeError,
    ApiValueError
)


class JobServiceApi(object):
    """NOTE: This class is auto generated by OpenAPI Generator
    Ref: https://openapi-generator.tech

    Do not edit the class manually.
    """

    def __init__(self, api_client=None):
        if api_client is None:
            api_client = ApiClient()
        self.api_client = api_client

    def job_service_create_job(self, job, **kwargs):  # noqa: E501
        """Creates a new job.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_create_job(job, async_req=True)
        >>> result = thread.get()

        :param job: The job to be created (required)
        :type job: V1Job
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: V1Job
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_create_job_with_http_info(job, **kwargs)  # noqa: E501

    def job_service_create_job_with_http_info(self, job, **kwargs):  # noqa: E501
        """Creates a new job.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_create_job_with_http_info(job, async_req=True)
        >>> result = thread.get()

        :param job: The job to be created (required)
        :type job: V1Job
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(V1Job, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'job'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_create_job" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']
        # verify the required parameter 'job' is set
        if self.api_client.client_side_validation and ('job' not in local_var_params or  # noqa: E501
                                                        local_var_params['job'] is None):  # noqa: E501
            raise ApiValueError("Missing the required parameter `job` when calling `job_service_create_job`")  # noqa: E501

        collection_formats = {}

        path_params = {}

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        if 'job' in local_var_params:
            body_params = local_var_params['job']
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # HTTP header `Content-Type`
        header_params['Content-Type'] = self.api_client.select_header_content_type(  # noqa: E501
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs', 'POST',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='V1Job',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)

    def job_service_delete_job(self, id, **kwargs):  # noqa: E501
        """Deletes a job.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_delete_job(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be deleted (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: object
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_delete_job_with_http_info(id, **kwargs)  # noqa: E501

    def job_service_delete_job_with_http_info(self, id, **kwargs):  # noqa: E501
        """Deletes a job.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_delete_job_with_http_info(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be deleted (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(object, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'id'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_delete_job" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']
        # verify the required parameter 'id' is set
        if self.api_client.client_side_validation and ('id' not in local_var_params or  # noqa: E501
                                                        local_var_params['id'] is None):  # noqa: E501
            raise ApiValueError("Missing the required parameter `id` when calling `job_service_delete_job`")  # noqa: E501

        collection_formats = {}

        path_params = {}
        if 'id' in local_var_params:
            path_params['id'] = local_var_params['id']  # noqa: E501

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs/{id}', 'DELETE',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='object',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)

    def job_service_disable_job(self, id, **kwargs):  # noqa: E501
        """Stops a job and all its associated runs. The job is not deleted.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_disable_job(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be disabled (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: object
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_disable_job_with_http_info(id, **kwargs)  # noqa: E501

    def job_service_disable_job_with_http_info(self, id, **kwargs):  # noqa: E501
        """Stops a job and all its associated runs. The job is not deleted.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_disable_job_with_http_info(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be disabled (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(object, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'id'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_disable_job" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']
        # verify the required parameter 'id' is set
        if self.api_client.client_side_validation and ('id' not in local_var_params or  # noqa: E501
                                                        local_var_params['id'] is None):  # noqa: E501
            raise ApiValueError("Missing the required parameter `id` when calling `job_service_disable_job`")  # noqa: E501

        collection_formats = {}

        path_params = {}
        if 'id' in local_var_params:
            path_params['id'] = local_var_params['id']  # noqa: E501

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs/{id}/disable', 'POST',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='object',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)

    def job_service_enable_job(self, id, **kwargs):  # noqa: E501
        """Restarts a job that was previously stopped. All runs associated with the job will continue.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_enable_job(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be enabled (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: object
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_enable_job_with_http_info(id, **kwargs)  # noqa: E501

    def job_service_enable_job_with_http_info(self, id, **kwargs):  # noqa: E501
        """Restarts a job that was previously stopped. All runs associated with the job will continue.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_enable_job_with_http_info(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be enabled (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(object, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'id'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_enable_job" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']
        # verify the required parameter 'id' is set
        if self.api_client.client_side_validation and ('id' not in local_var_params or  # noqa: E501
                                                        local_var_params['id'] is None):  # noqa: E501
            raise ApiValueError("Missing the required parameter `id` when calling `job_service_enable_job`")  # noqa: E501

        collection_formats = {}

        path_params = {}
        if 'id' in local_var_params:
            path_params['id'] = local_var_params['id']  # noqa: E501

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs/{id}/enable', 'POST',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='object',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)

    def job_service_get_job(self, id, **kwargs):  # noqa: E501
        """Finds a specific job by ID.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_get_job(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be retrieved (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: V1Job
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_get_job_with_http_info(id, **kwargs)  # noqa: E501

    def job_service_get_job_with_http_info(self, id, **kwargs):  # noqa: E501
        """Finds a specific job by ID.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_get_job_with_http_info(id, async_req=True)
        >>> result = thread.get()

        :param id: The ID of the job to be retrieved (required)
        :type id: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(V1Job, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'id'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_get_job" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']
        # verify the required parameter 'id' is set
        if self.api_client.client_side_validation and ('id' not in local_var_params or  # noqa: E501
                                                        local_var_params['id'] is None):  # noqa: E501
            raise ApiValueError("Missing the required parameter `id` when calling `job_service_get_job`")  # noqa: E501

        collection_formats = {}

        path_params = {}
        if 'id' in local_var_params:
            path_params['id'] = local_var_params['id']  # noqa: E501

        query_params = []

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs/{id}', 'GET',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='V1Job',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)

    def job_service_list_jobs(self, **kwargs):  # noqa: E501
        """Finds all jobs.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_list_jobs(async_req=True)
        >>> result = thread.get()

        :param page_token: A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListJobs call or can be omitted when fetching the first page.
        :type page_token: str
        :param page_size: The number of jobs to be listed per page. If there are more jobs than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
        :type page_size: int
        :param sort_by: Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default.
        :type sort_by: str
        :param resource_reference_key_type: The type of the resource that referred to.
        :type resource_reference_key_type: str
        :param resource_reference_key_id: The ID of the resource that referred to.
        :type resource_reference_key_id: str
        :param filter: A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1/filter.proto)).
        :type filter: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: V1ListJobsResponse
        """
        kwargs['_return_http_data_only'] = True
        return self.job_service_list_jobs_with_http_info(**kwargs)  # noqa: E501

    def job_service_list_jobs_with_http_info(self, **kwargs):  # noqa: E501
        """Finds all jobs.  # noqa: E501

        This method makes a synchronous HTTP request by default. To make an
        asynchronous HTTP request, please pass async_req=True

        >>> thread = api.job_service_list_jobs_with_http_info(async_req=True)
        >>> result = thread.get()

        :param page_token: A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListJobs call or can be omitted when fetching the first page.
        :type page_token: str
        :param page_size: The number of jobs to be listed per page. If there are more jobs than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
        :type page_size: int
        :param sort_by: Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default.
        :type sort_by: str
        :param resource_reference_key_type: The type of the resource that referred to.
        :type resource_reference_key_type: str
        :param resource_reference_key_id: The ID of the resource that referred to.
        :type resource_reference_key_id: str
        :param filter: A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1/filter.proto)).
        :type filter: str
        :param async_req: Whether to execute the request asynchronously.
        :type async_req: bool, optional
        :param _return_http_data_only: response data without head status code
                                       and headers
        :type _return_http_data_only: bool, optional
        :param _preload_content: if False, the urllib3.HTTPResponse object will
                                 be returned without reading/decoding response
                                 data. Default is True.
        :type _preload_content: bool, optional
        :param _request_timeout: timeout setting for this request. If one
                                 number provided, it will be total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :return: Returns the result object.
                 If the method is called asynchronously,
                 returns the request thread.
        :rtype: tuple(V1ListJobsResponse, status_code(int), headers(HTTPHeaderDict))
        """

        local_var_params = locals()

        all_params = [
            'page_token',
            'page_size',
            'sort_by',
            'resource_reference_key_type',
            'resource_reference_key_id',
            'filter'
        ]
        all_params.extend(
            [
                'async_req',
                '_return_http_data_only',
                '_preload_content',
                '_request_timeout'
            ]
        )

        for key, val in six.iteritems(local_var_params['kwargs']):
            if key not in all_params:
                raise ApiTypeError(
                    "Got an unexpected keyword argument '%s'"
                    " to method job_service_list_jobs" % key
                )
            local_var_params[key] = val
        del local_var_params['kwargs']

        collection_formats = {}

        path_params = {}

        query_params = []
        if 'page_token' in local_var_params and local_var_params['page_token'] is not None:  # noqa: E501
            query_params.append(('pageToken', local_var_params['page_token']))  # noqa: E501
        if 'page_size' in local_var_params and local_var_params['page_size'] is not None:  # noqa: E501
            query_params.append(('pageSize', local_var_params['page_size']))  # noqa: E501
        if 'sort_by' in local_var_params and local_var_params['sort_by'] is not None:  # noqa: E501
            query_params.append(('sortBy', local_var_params['sort_by']))  # noqa: E501
        if 'resource_reference_key_type' in local_var_params and local_var_params['resource_reference_key_type'] is not None:  # noqa: E501
            query_params.append(('resourceReferenceKey.type', local_var_params['resource_reference_key_type']))  # noqa: E501
        if 'resource_reference_key_id' in local_var_params and local_var_params['resource_reference_key_id'] is not None:  # noqa: E501
            query_params.append(('resourceReferenceKey.id', local_var_params['resource_reference_key_id']))  # noqa: E501
        if 'filter' in local_var_params and local_var_params['filter'] is not None:  # noqa: E501
            query_params.append(('filter', local_var_params['filter']))  # noqa: E501

        header_params = {}

        form_params = []
        local_var_files = {}

        body_params = None
        # HTTP header `Accept`
        header_params['Accept'] = self.api_client.select_header_accept(
            ['application/json'])  # noqa: E501

        # Authentication setting
        auth_settings = ['Bearer']  # noqa: E501

        return self.api_client.call_api(
            '/apis/v1/jobs', 'GET',
            path_params,
            query_params,
            header_params,
            body=body_params,
            post_params=form_params,
            files=local_var_files,
            response_type='V1ListJobsResponse',  # noqa: E501
            auth_settings=auth_settings,
            async_req=local_var_params.get('async_req'),
            _return_http_data_only=local_var_params.get('_return_http_data_only'),  # noqa: E501
            _preload_content=local_var_params.get('_preload_content', True),
            _request_timeout=local_var_params.get('_request_timeout'),
            collection_formats=collection_formats)
