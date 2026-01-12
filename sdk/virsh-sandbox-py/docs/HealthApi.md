# virsh_sandbox.HealthApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_health**](HealthApi.md#get_health) | **GET** /v1/health | Health check


# **get_health**
> VirshSandboxInternalRestHealthResponse get_health()

Health check

Returns service health status

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.virsh_sandbox_internal_rest_health_response import VirshSandboxInternalRestHealthResponse
from virsh_sandbox.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = virsh_sandbox.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with virsh_sandbox.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = virsh_sandbox.HealthApi(api_client)

    try:
        # Health check
        api_response = api_instance.get_health()
        print("The response of HealthApi->get_health:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HealthApi->get_health: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**VirshSandboxInternalRestHealthResponse**](VirshSandboxInternalRestHealthResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

