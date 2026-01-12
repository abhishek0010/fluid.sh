# VirshSandboxInternalRestListSandboxCommandsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[VirshSandboxInternalStoreCommand]**](VirshSandboxInternalStoreCommand.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_list_sandbox_commands_response import VirshSandboxInternalRestListSandboxCommandsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestListSandboxCommandsResponse from a JSON string
virsh_sandbox_internal_rest_list_sandbox_commands_response_instance = VirshSandboxInternalRestListSandboxCommandsResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestListSandboxCommandsResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_list_sandbox_commands_response_dict = virsh_sandbox_internal_rest_list_sandbox_commands_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestListSandboxCommandsResponse from a dict
virsh_sandbox_internal_rest_list_sandbox_commands_response_from_dict = VirshSandboxInternalRestListSandboxCommandsResponse.from_dict(virsh_sandbox_internal_rest_list_sandbox_commands_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


