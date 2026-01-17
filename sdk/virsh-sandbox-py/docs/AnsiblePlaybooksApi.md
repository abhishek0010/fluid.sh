# virsh_sandbox.AnsiblePlaybooksApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**add_playbook_task**](AnsiblePlaybooksApi.md#add_playbook_task) | **POST** /v1/ansible/playbooks/{playbook_name}/tasks | Add task to playbook
[**create_playbook**](AnsiblePlaybooksApi.md#create_playbook) | **POST** /v1/ansible/playbooks | Create playbook
[**delete_playbook**](AnsiblePlaybooksApi.md#delete_playbook) | **DELETE** /v1/ansible/playbooks/{playbook_name} | Delete playbook
[**delete_playbook_task**](AnsiblePlaybooksApi.md#delete_playbook_task) | **DELETE** /v1/ansible/playbooks/{playbook_name}/tasks/{task_id} | Delete task
[**export_playbook**](AnsiblePlaybooksApi.md#export_playbook) | **GET** /v1/ansible/playbooks/{playbook_name}/export | Export playbook
[**get_playbook**](AnsiblePlaybooksApi.md#get_playbook) | **GET** /v1/ansible/playbooks/{playbook_name} | Get playbook
[**list_playbooks**](AnsiblePlaybooksApi.md#list_playbooks) | **GET** /v1/ansible/playbooks | List playbooks
[**reorder_playbook_tasks**](AnsiblePlaybooksApi.md#reorder_playbook_tasks) | **PATCH** /v1/ansible/playbooks/{playbook_name}/tasks/reorder | Reorder tasks
[**update_playbook_task**](AnsiblePlaybooksApi.md#update_playbook_task) | **PUT** /v1/ansible/playbooks/{playbook_name}/tasks/{task_id} | Update task


# **add_playbook_task**
> InternalAnsibleAddTaskResponse add_playbook_task(playbook_name, request)

Add task to playbook

Adds a new task to an existing playbook

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_add_task_request import InternalAnsibleAddTaskRequest
from virsh_sandbox.models.internal_ansible_add_task_response import InternalAnsibleAddTaskResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name
    request = virsh_sandbox.InternalAnsibleAddTaskRequest() # InternalAnsibleAddTaskRequest | Task parameters

    try:
        # Add task to playbook
        api_response = api_instance.add_playbook_task(playbook_name, request)
        print("The response of AnsiblePlaybooksApi->add_playbook_task:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->add_playbook_task: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 
 **request** | [**InternalAnsibleAddTaskRequest**](InternalAnsibleAddTaskRequest.md)| Task parameters | 

### Return type

[**InternalAnsibleAddTaskResponse**](InternalAnsibleAddTaskResponse.md)

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
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_playbook**
> InternalAnsibleCreatePlaybookResponse create_playbook(request)

Create playbook

Creates a new Ansible playbook

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_create_playbook_request import InternalAnsibleCreatePlaybookRequest
from virsh_sandbox.models.internal_ansible_create_playbook_response import InternalAnsibleCreatePlaybookResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    request = virsh_sandbox.InternalAnsibleCreatePlaybookRequest() # InternalAnsibleCreatePlaybookRequest | Playbook creation parameters

    try:
        # Create playbook
        api_response = api_instance.create_playbook(request)
        print("The response of AnsiblePlaybooksApi->create_playbook:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->create_playbook: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**InternalAnsibleCreatePlaybookRequest**](InternalAnsibleCreatePlaybookRequest.md)| Playbook creation parameters | 

### Return type

[**InternalAnsibleCreatePlaybookResponse**](InternalAnsibleCreatePlaybookResponse.md)

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
**409** | Conflict |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_playbook**
> delete_playbook(playbook_name)

Delete playbook

Deletes a playbook and all its tasks

### Example


```python
import virsh_sandbox
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name

    try:
        # Delete playbook
        api_instance.delete_playbook(playbook_name)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->delete_playbook: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 

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
**204** | No Content |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_playbook_task**
> delete_playbook_task(playbook_name, task_id)

Delete task

Removes a task from a playbook

### Example


```python
import virsh_sandbox
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name
    task_id = 'task_id_example' # str | Task ID

    try:
        # Delete task
        api_instance.delete_playbook_task(playbook_name, task_id)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->delete_playbook_task: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 
 **task_id** | **str**| Task ID | 

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
**204** | No Content |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **export_playbook**
> InternalAnsibleExportPlaybookResponse export_playbook(playbook_name)

Export playbook

Exports a playbook as raw YAML

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_export_playbook_response import InternalAnsibleExportPlaybookResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name

    try:
        # Export playbook
        api_response = api_instance.export_playbook(playbook_name)
        print("The response of AnsiblePlaybooksApi->export_playbook:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->export_playbook: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 

### Return type

[**InternalAnsibleExportPlaybookResponse**](InternalAnsibleExportPlaybookResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_playbook**
> InternalAnsibleGetPlaybookResponse get_playbook(playbook_name)

Get playbook

Gets a playbook and its tasks by name

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_get_playbook_response import InternalAnsibleGetPlaybookResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name

    try:
        # Get playbook
        api_response = api_instance.get_playbook(playbook_name)
        print("The response of AnsiblePlaybooksApi->get_playbook:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->get_playbook: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 

### Return type

[**InternalAnsibleGetPlaybookResponse**](InternalAnsibleGetPlaybookResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_playbooks**
> InternalAnsibleListPlaybooksResponse list_playbooks()

List playbooks

Lists all Ansible playbooks

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_list_playbooks_response import InternalAnsibleListPlaybooksResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)

    try:
        # List playbooks
        api_response = api_instance.list_playbooks()
        print("The response of AnsiblePlaybooksApi->list_playbooks:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->list_playbooks: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**InternalAnsibleListPlaybooksResponse**](InternalAnsibleListPlaybooksResponse.md)

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

# **reorder_playbook_tasks**
> reorder_playbook_tasks(playbook_name, request)

Reorder tasks

Reorders tasks in a playbook

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_reorder_tasks_request import InternalAnsibleReorderTasksRequest
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name
    request = virsh_sandbox.InternalAnsibleReorderTasksRequest() # InternalAnsibleReorderTasksRequest | New task order

    try:
        # Reorder tasks
        api_instance.reorder_playbook_tasks(playbook_name, request)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->reorder_playbook_tasks: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 
 **request** | [**InternalAnsibleReorderTasksRequest**](InternalAnsibleReorderTasksRequest.md)| New task order | 

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
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_playbook_task**
> InternalAnsibleUpdateTaskResponse update_playbook_task(playbook_name, task_id, request)

Update task

Updates an existing task in a playbook

### Example


```python
import virsh_sandbox
from virsh_sandbox.models.internal_ansible_update_task_request import InternalAnsibleUpdateTaskRequest
from virsh_sandbox.models.internal_ansible_update_task_response import InternalAnsibleUpdateTaskResponse
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
    api_instance = virsh_sandbox.AnsiblePlaybooksApi(api_client)
    playbook_name = 'playbook_name_example' # str | Playbook name
    task_id = 'task_id_example' # str | Task ID
    request = virsh_sandbox.InternalAnsibleUpdateTaskRequest() # InternalAnsibleUpdateTaskRequest | Task update parameters

    try:
        # Update task
        api_response = api_instance.update_playbook_task(playbook_name, task_id, request)
        print("The response of AnsiblePlaybooksApi->update_playbook_task:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AnsiblePlaybooksApi->update_playbook_task: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **playbook_name** | **str**| Playbook name | 
 **task_id** | **str**| Task ID | 
 **request** | [**InternalAnsibleUpdateTaskRequest**](InternalAnsibleUpdateTaskRequest.md)| Task update parameters | 

### Return type

[**InternalAnsibleUpdateTaskResponse**](InternalAnsibleUpdateTaskResponse.md)

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
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

