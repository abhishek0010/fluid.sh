# VirshSandboxInternalRestGetSandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[VirshSandboxInternalStoreCommand]**](VirshSandboxInternalStoreCommand.md) |  | [optional] 
**sandbox** | [**VirshSandboxInternalStoreSandbox**](VirshSandboxInternalStoreSandbox.md) |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_get_sandbox_response import VirshSandboxInternalRestGetSandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestGetSandboxResponse from a JSON string
virsh_sandbox_internal_rest_get_sandbox_response_instance = VirshSandboxInternalRestGetSandboxResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestGetSandboxResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_get_sandbox_response_dict = virsh_sandbox_internal_rest_get_sandbox_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestGetSandboxResponse from a dict
virsh_sandbox_internal_rest_get_sandbox_response_from_dict = VirshSandboxInternalRestGetSandboxResponse.from_dict(virsh_sandbox_internal_rest_get_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


