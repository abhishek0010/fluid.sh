# fluid.SandboxApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_sandbox**](SandboxApi.md#create_sandbox) | **POST** /v1/sandboxes | Create a new sandbox
[**create_snapshot**](SandboxApi.md#create_snapshot) | **POST** /v1/sandboxes/{id}/snapshot | Create snapshot
[**destroy_sandbox**](SandboxApi.md#destroy_sandbox) | **DELETE** /v1/sandboxes/{id} | Destroy sandbox
[**diff_snapshots**](SandboxApi.md#diff_snapshots) | **POST** /v1/sandboxes/{id}/diff | Diff snapshots
[**discover_sandbox_ip**](SandboxApi.md#discover_sandbox_ip) | **GET** /v1/sandboxes/{id}/ip | Discover sandbox IP
[**generate_configuration**](SandboxApi.md#generate_configuration) | **POST** /v1/sandboxes/{id}/generate/{tool} | Generate configuration
[**get_sandbox**](SandboxApi.md#get_sandbox) | **GET** /v1/sandboxes/{id} | Get sandbox details
[**inject_ssh_key**](SandboxApi.md#inject_ssh_key) | **POST** /v1/sandboxes/{id}/sshkey | Inject SSH key into sandbox
[**list_sandbox_commands**](SandboxApi.md#list_sandbox_commands) | **GET** /v1/sandboxes/{id}/commands | List sandbox commands
[**list_sandboxes**](SandboxApi.md#list_sandboxes) | **GET** /v1/sandboxes | List sandboxes
[**publish_changes**](SandboxApi.md#publish_changes) | **POST** /v1/sandboxes/{id}/publish | Publish changes
[**run_sandbox_command**](SandboxApi.md#run_sandbox_command) | **POST** /v1/sandboxes/{id}/run | Run command in sandbox
[**start_sandbox**](SandboxApi.md#start_sandbox) | **POST** /v1/sandboxes/{id}/start | Start sandbox
[**stream_sandbox_activity**](SandboxApi.md#stream_sandbox_activity) | **GET** /v1/sandboxes/{id}/stream | Stream sandbox activity


# **create_sandbox**
> FluidRemoteInternalRestCreateSandboxResponse create_sandbox(request)

Create a new sandbox

Creates a new virtual machine sandbox by cloning from an existing VM

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_create_sandbox_request import FluidRemoteInternalRestCreateSandboxRequest
from fluid.models.fluid_remote_internal_rest_create_sandbox_response import FluidRemoteInternalRestCreateSandboxResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    request = fluid.FluidRemoteInternalRestCreateSandboxRequest() # FluidRemoteInternalRestCreateSandboxRequest | Sandbox creation parameters

    try:
        # Create a new sandbox
        api_response = api_instance.create_sandbox(request)
        print("The response of SandboxApi->create_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->create_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**FluidRemoteInternalRestCreateSandboxRequest**](FluidRemoteInternalRestCreateSandboxRequest.md)| Sandbox creation parameters | 

### Return type

[**FluidRemoteInternalRestCreateSandboxResponse**](FluidRemoteInternalRestCreateSandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Created |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_snapshot**
> FluidRemoteInternalRestSnapshotResponse create_snapshot(id, request)

Create snapshot

Creates a snapshot of the sandbox

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_snapshot_request import FluidRemoteInternalRestSnapshotRequest
from fluid.models.fluid_remote_internal_rest_snapshot_response import FluidRemoteInternalRestSnapshotResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestSnapshotRequest() # FluidRemoteInternalRestSnapshotRequest | Snapshot parameters

    try:
        # Create snapshot
        api_response = api_instance.create_snapshot(id, request)
        print("The response of SandboxApi->create_snapshot:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->create_snapshot: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestSnapshotRequest**](FluidRemoteInternalRestSnapshotRequest.md)| Snapshot parameters | 

### Return type

[**FluidRemoteInternalRestSnapshotResponse**](FluidRemoteInternalRestSnapshotResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Created |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **destroy_sandbox**
> FluidRemoteInternalRestDestroySandboxResponse destroy_sandbox(id)

Destroy sandbox

Destroys the sandbox and cleans up resources

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_destroy_sandbox_response import FluidRemoteInternalRestDestroySandboxResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Destroy sandbox
        api_response = api_instance.destroy_sandbox(id)
        print("The response of SandboxApi->destroy_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->destroy_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

[**FluidRemoteInternalRestDestroySandboxResponse**](FluidRemoteInternalRestDestroySandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **diff_snapshots**
> FluidRemoteInternalRestDiffResponse diff_snapshots(id, request)

Diff snapshots

Computes differences between two snapshots

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_diff_request import FluidRemoteInternalRestDiffRequest
from fluid.models.fluid_remote_internal_rest_diff_response import FluidRemoteInternalRestDiffResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestDiffRequest() # FluidRemoteInternalRestDiffRequest | Diff parameters

    try:
        # Diff snapshots
        api_response = api_instance.diff_snapshots(id, request)
        print("The response of SandboxApi->diff_snapshots:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->diff_snapshots: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestDiffRequest**](FluidRemoteInternalRestDiffRequest.md)| Diff parameters | 

### Return type

[**FluidRemoteInternalRestDiffResponse**](FluidRemoteInternalRestDiffResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **discover_sandbox_ip**
> FluidRemoteInternalRestDiscoverIPResponse discover_sandbox_ip(id)

Discover sandbox IP

Discovers and returns the IP address for a running sandbox. Use this for async workflows where wait_for_ip was false during start.

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_discover_ip_response import FluidRemoteInternalRestDiscoverIPResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Discover sandbox IP
        api_response = api_instance.discover_sandbox_ip(id)
        print("The response of SandboxApi->discover_sandbox_ip:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->discover_sandbox_ip: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

[**FluidRemoteInternalRestDiscoverIPResponse**](FluidRemoteInternalRestDiscoverIPResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **generate_configuration**
> generate_configuration(id, tool)

Generate configuration

Generates Ansible or Puppet configuration from sandbox changes

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    tool = 'tool_example' # str | Tool type (ansible or puppet)

    try:
        # Generate configuration
        api_instance.generate_configuration(id, tool)
    except Exception as e:
        print("Exception when calling SandboxApi->generate_configuration: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **tool** | **str**| Tool type (ansible or puppet) | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**400** | Bad Request |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_sandbox**
> FluidRemoteInternalRestGetSandboxResponse get_sandbox(id, include_commands=include_commands)

Get sandbox details

Returns detailed information about a specific sandbox including recent commands

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_get_sandbox_response import FluidRemoteInternalRestGetSandboxResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    include_commands = True # bool | Include command history (optional)

    try:
        # Get sandbox details
        api_response = api_instance.get_sandbox(id, include_commands=include_commands)
        print("The response of SandboxApi->get_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->get_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **include_commands** | **bool**| Include command history | [optional] 

### Return type

[**FluidRemoteInternalRestGetSandboxResponse**](FluidRemoteInternalRestGetSandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **inject_ssh_key**
> inject_ssh_key(id, request)

Inject SSH key into sandbox

Injects a public SSH key for a user in the sandbox

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_inject_ssh_key_request import FluidRemoteInternalRestInjectSSHKeyRequest
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestInjectSSHKeyRequest() # FluidRemoteInternalRestInjectSSHKeyRequest | SSH key injection parameters

    try:
        # Inject SSH key into sandbox
        api_instance.inject_ssh_key(id, request)
    except Exception as e:
        print("Exception when calling SandboxApi->inject_ssh_key: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestInjectSSHKeyRequest**](FluidRemoteInternalRestInjectSSHKeyRequest.md)| SSH key injection parameters | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**204** | No Content |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_sandbox_commands**
> FluidRemoteInternalRestListSandboxCommandsResponse list_sandbox_commands(id, limit=limit, offset=offset)

List sandbox commands

Returns all commands executed in the sandbox

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_list_sandbox_commands_response import FluidRemoteInternalRestListSandboxCommandsResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    limit = 56 # int | Max results to return (optional)
    offset = 56 # int | Number of results to skip (optional)

    try:
        # List sandbox commands
        api_response = api_instance.list_sandbox_commands(id, limit=limit, offset=offset)
        print("The response of SandboxApi->list_sandbox_commands:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->list_sandbox_commands: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **limit** | **int**| Max results to return | [optional] 
 **offset** | **int**| Number of results to skip | [optional] 

### Return type

[**FluidRemoteInternalRestListSandboxCommandsResponse**](FluidRemoteInternalRestListSandboxCommandsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_sandboxes**
> FluidRemoteInternalRestListSandboxesResponse list_sandboxes(agent_id=agent_id, job_id=job_id, base_image=base_image, state=state, vm_name=vm_name, limit=limit, offset=offset)

List sandboxes

Lists all sandboxes with optional filtering by agent_id, job_id, base_image, state, or vm_name

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_list_sandboxes_response import FluidRemoteInternalRestListSandboxesResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    agent_id = 'agent_id_example' # str | Filter by agent ID (optional)
    job_id = 'job_id_example' # str | Filter by job ID (optional)
    base_image = 'base_image_example' # str | Filter by base image (optional)
    state = 'state_example' # str | Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) (optional)
    vm_name = 'vm_name_example' # str | Filter by VM name (optional)
    limit = 56 # int | Max results to return (optional)
    offset = 56 # int | Number of results to skip (optional)

    try:
        # List sandboxes
        api_response = api_instance.list_sandboxes(agent_id=agent_id, job_id=job_id, base_image=base_image, state=state, vm_name=vm_name, limit=limit, offset=offset)
        print("The response of SandboxApi->list_sandboxes:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->list_sandboxes: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **agent_id** | **str**| Filter by agent ID | [optional] 
 **job_id** | **str**| Filter by job ID | [optional] 
 **base_image** | **str**| Filter by base image | [optional] 
 **state** | **str**| Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) | [optional] 
 **vm_name** | **str**| Filter by VM name | [optional] 
 **limit** | **int**| Max results to return | [optional] 
 **offset** | **int**| Number of results to skip | [optional] 

### Return type

[**FluidRemoteInternalRestListSandboxesResponse**](FluidRemoteInternalRestListSandboxesResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **publish_changes**
> publish_changes(id, request)

Publish changes

Publishes sandbox changes to GitOps repository

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_publish_request import FluidRemoteInternalRestPublishRequest
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestPublishRequest() # FluidRemoteInternalRestPublishRequest | Publish parameters

    try:
        # Publish changes
        api_instance.publish_changes(id, request)
    except Exception as e:
        print("Exception when calling SandboxApi->publish_changes: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestPublishRequest**](FluidRemoteInternalRestPublishRequest.md)| Publish parameters | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**400** | Bad Request |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_sandbox_command**
> FluidRemoteInternalRestRunCommandResponse run_sandbox_command(id, request)

Run command in sandbox

Executes a command inside the sandbox via SSH. If private_key_path is omitted and SSH CA is configured, managed credentials will be used automatically.

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_run_command_request import FluidRemoteInternalRestRunCommandRequest
from fluid.models.fluid_remote_internal_rest_run_command_response import FluidRemoteInternalRestRunCommandResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestRunCommandRequest() # FluidRemoteInternalRestRunCommandRequest | Command execution parameters

    try:
        # Run command in sandbox
        api_response = api_instance.run_sandbox_command(id, request)
        print("The response of SandboxApi->run_sandbox_command:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->run_sandbox_command: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestRunCommandRequest**](FluidRemoteInternalRestRunCommandRequest.md)| Command execution parameters | 

### Return type

[**FluidRemoteInternalRestRunCommandResponse**](FluidRemoteInternalRestRunCommandResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **start_sandbox**
> FluidRemoteInternalRestStartSandboxResponse start_sandbox(id, request=request)

Start sandbox

Starts the virtual machine sandbox

### Example


```python
import fluid
from fluid.models.fluid_remote_internal_rest_start_sandbox_request import FluidRemoteInternalRestStartSandboxRequest
from fluid.models.fluid_remote_internal_rest_start_sandbox_response import FluidRemoteInternalRestStartSandboxResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.FluidRemoteInternalRestStartSandboxRequest() # FluidRemoteInternalRestStartSandboxRequest | Start parameters (optional)

    try:
        # Start sandbox
        api_response = api_instance.start_sandbox(id, request=request)
        print("The response of SandboxApi->start_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->start_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**FluidRemoteInternalRestStartSandboxRequest**](FluidRemoteInternalRestStartSandboxRequest.md)| Start parameters | [optional] 

### Return type

[**FluidRemoteInternalRestStartSandboxResponse**](FluidRemoteInternalRestStartSandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **stream_sandbox_activity**
> stream_sandbox_activity(id)

Stream sandbox activity

Connects via WebSocket to stream realtime sandbox activity (commands, file changes)

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Stream sandbox activity
        api_instance.stream_sandbox_activity(id)
    except Exception as e:
        print("Exception when calling SandboxApi->stream_sandbox_activity: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**101** | Switching Protocols - WebSocket connection established |  -  |
**400** | Invalid sandbox ID |  -  |
**404** | Sandbox not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

