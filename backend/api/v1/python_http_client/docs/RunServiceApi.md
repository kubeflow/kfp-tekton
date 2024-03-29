# kfp_tekton_server_api.RunServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**run_service_archive_run**](RunServiceApi.md#run_service_archive_run) | **POST** /apis/v1/runs/{id}:archive | Archives a run.
[**run_service_create_run**](RunServiceApi.md#run_service_create_run) | **POST** /apis/v1/runs | Creates a new run.
[**run_service_delete_run**](RunServiceApi.md#run_service_delete_run) | **DELETE** /apis/v1/runs/{id} | Deletes a run.
[**run_service_get_run**](RunServiceApi.md#run_service_get_run) | **GET** /apis/v1/runs/{run_id} | Finds a specific run by ID.
[**run_service_list_runs**](RunServiceApi.md#run_service_list_runs) | **GET** /apis/v1/runs | Finds all runs.
[**run_service_read_artifact**](RunServiceApi.md#run_service_read_artifact) | **GET** /apis/v1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read | Finds a run&#39;s artifact data.
[**run_service_report_run_metrics**](RunServiceApi.md#run_service_report_run_metrics) | **POST** /apis/v1/runs/{run_id}:reportMetrics | ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.
[**run_service_retry_run**](RunServiceApi.md#run_service_retry_run) | **POST** /apis/v1/runs/{run_id}/retry | Re-initiates a failed or terminated run.
[**run_service_terminate_run**](RunServiceApi.md#run_service_terminate_run) | **POST** /apis/v1/runs/{run_id}/terminate | Terminates an active run.
[**run_service_unarchive_run**](RunServiceApi.md#run_service_unarchive_run) | **POST** /apis/v1/runs/{id}:unarchive | Restores an archived run.


# **run_service_archive_run**
> object run_service_archive_run(id)

Archives a run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    id = 'id_example' # str | The ID of the run to be archived.

    try:
        # Archives a run.
        api_response = api_instance.run_service_archive_run(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_archive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the run to be archived. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_create_run**
> V1RunDetail run_service_create_run(run)

Creates a new run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run = kfp_tekton_server_api.V1Run() # V1Run | 

    try:
        # Creates a new run.
        api_response = api_instance.run_service_create_run(run)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_create_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run** | [**V1Run**](V1Run.md)|  | 

### Return type

[**V1RunDetail**](V1RunDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_delete_run**
> object run_service_delete_run(id)

Deletes a run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    id = 'id_example' # str | The ID of the run to be deleted.

    try:
        # Deletes a run.
        api_response = api_instance.run_service_delete_run(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_delete_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the run to be deleted. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_get_run**
> V1RunDetail run_service_get_run(run_id)

Finds a specific run by ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run_id = 'run_id_example' # str | The ID of the run to be retrieved.

    try:
        # Finds a specific run by ID.
        api_response = api_instance.run_service_get_run(run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_get_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be retrieved. | 

### Return type

[**V1RunDetail**](V1RunDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_list_runs**
> V1ListRunsResponse run_service_list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id, filter=filter)

Finds all runs.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" (Example, \"name asc\" or \"id desc\"). Ascending by default. (optional)
resource_reference_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to 'UNKNOWN_RESOURCE_TYPE')
resource_reference_key_id = 'resource_reference_key_id_example' # str | The ID of the resource that referred to. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1/filter.proto)). (optional)

    try:
        # Finds all runs.
        api_response = api_instance.run_service_list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_list_runs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; (Example, \&quot;name asc\&quot; or \&quot;id desc\&quot;). Ascending by default. | [optional] 
 **resource_reference_key_type** | **str**| The type of the resource that referred to. | [optional] [default to &#39;UNKNOWN_RESOURCE_TYPE&#39;]
 **resource_reference_key_id** | **str**| The ID of the resource that referred to. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1/filter.proto)). | [optional] 

### Return type

[**V1ListRunsResponse**](V1ListRunsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_read_artifact**
> V1ReadArtifactResponse run_service_read_artifact(run_id, node_id, artifact_name)

Finds a run's artifact data.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run_id = 'run_id_example' # str | The ID of the run.
node_id = 'node_id_example' # str | The ID of the running node.
artifact_name = 'artifact_name_example' # str | The name of the artifact.

    try:
        # Finds a run's artifact data.
        api_response = api_instance.run_service_read_artifact(run_id, node_id, artifact_name)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_read_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run. | 
 **node_id** | **str**| The ID of the running node. | 
 **artifact_name** | **str**| The name of the artifact. | 

### Return type

[**V1ReadArtifactResponse**](V1ReadArtifactResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_report_run_metrics**
> V1ReportRunMetricsResponse run_service_report_run_metrics(run_id, body)

ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run_id = 'run_id_example' # str | Required. The parent run ID of the metric.
body = kfp_tekton_server_api.InlineObject() # InlineObject | 

    try:
        # ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.
        api_response = api_instance.run_service_report_run_metrics(run_id, body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_report_run_metrics: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| Required. The parent run ID of the metric. | 
 **body** | [**InlineObject**](InlineObject.md)|  | 

### Return type

[**V1ReportRunMetricsResponse**](V1ReportRunMetricsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_retry_run**
> object run_service_retry_run(run_id)

Re-initiates a failed or terminated run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run_id = 'run_id_example' # str | The ID of the run to be retried.

    try:
        # Re-initiates a failed or terminated run.
        api_response = api_instance.run_service_retry_run(run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_retry_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be retried. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_terminate_run**
> object run_service_terminate_run(run_id)

Terminates an active run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    run_id = 'run_id_example' # str | The ID of the run to be terminated.

    try:
        # Terminates an active run.
        api_response = api_instance.run_service_terminate_run(run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_terminate_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be terminated. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_unarchive_run**
> object run_service_unarchive_run(id)

Restores an archived run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_tekton_server_api
from kfp_tekton_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_tekton_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_tekton_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_tekton_server_api.RunServiceApi(api_client)
    id = 'id_example' # str | The ID of the run to be restored.

    try:
        # Restores an archived run.
        api_response = api_instance.run_service_unarchive_run(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_unarchive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the run to be restored. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

