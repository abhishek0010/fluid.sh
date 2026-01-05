# TmuxClientInternalApiListSandboxSessionsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sessions** | [**List[TmuxClientInternalApiSandboxSessionInfo]**](TmuxClientInternalApiSandboxSessionInfo.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from virsh_sandbox.models.tmux_client_internal_api_list_sandbox_sessions_response import TmuxClientInternalApiListSandboxSessionsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of TmuxClientInternalApiListSandboxSessionsResponse from a JSON string
tmux_client_internal_api_list_sandbox_sessions_response_instance = TmuxClientInternalApiListSandboxSessionsResponse.from_json(json)
# print the JSON string representation of the object
print(TmuxClientInternalApiListSandboxSessionsResponse.to_json())

# convert the object into a dict
tmux_client_internal_api_list_sandbox_sessions_response_dict = tmux_client_internal_api_list_sandbox_sessions_response_instance.to_dict()
# create an instance of TmuxClientInternalApiListSandboxSessionsResponse from a dict
tmux_client_internal_api_list_sandbox_sessions_response_from_dict = TmuxClientInternalApiListSandboxSessionsResponse.from_dict(tmux_client_internal_api_list_sandbox_sessions_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


