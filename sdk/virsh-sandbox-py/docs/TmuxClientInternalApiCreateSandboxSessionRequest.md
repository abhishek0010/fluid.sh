# TmuxClientInternalApiCreateSandboxSessionRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sandbox_id** | **str** | SandboxID is the ID of the sandbox to connect to | [optional] 
**session_name** | **str** | SessionName is the optional tmux session name (auto-generated if empty) | [optional] 
**ttl_minutes** | **int** | TTLMinutes is the certificate TTL in minutes (1-10, default 5) | [optional] 

## Example

```python
from virsh_sandbox.models.tmux_client_internal_api_create_sandbox_session_request import TmuxClientInternalApiCreateSandboxSessionRequest

# TODO update the JSON string below
json = "{}"
# create an instance of TmuxClientInternalApiCreateSandboxSessionRequest from a JSON string
tmux_client_internal_api_create_sandbox_session_request_instance = TmuxClientInternalApiCreateSandboxSessionRequest.from_json(json)
# print the JSON string representation of the object
print(TmuxClientInternalApiCreateSandboxSessionRequest.to_json())

# convert the object into a dict
tmux_client_internal_api_create_sandbox_session_request_dict = tmux_client_internal_api_create_sandbox_session_request_instance.to_dict()
# create an instance of TmuxClientInternalApiCreateSandboxSessionRequest from a dict
tmux_client_internal_api_create_sandbox_session_request_from_dict = TmuxClientInternalApiCreateSandboxSessionRequest.from_dict(tmux_client_internal_api_create_sandbox_session_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


