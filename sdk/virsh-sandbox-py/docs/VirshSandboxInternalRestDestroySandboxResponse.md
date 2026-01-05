# VirshSandboxInternalRestDestroySandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**base_image** | **str** |  | [optional] 
**sandbox_name** | **str** |  | [optional] 
**state** | [**VirshSandboxInternalStoreSandboxState**](VirshSandboxInternalStoreSandboxState.md) |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_destroy_sandbox_response import VirshSandboxInternalRestDestroySandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestDestroySandboxResponse from a JSON string
virsh_sandbox_internal_rest_destroy_sandbox_response_instance = VirshSandboxInternalRestDestroySandboxResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestDestroySandboxResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_destroy_sandbox_response_dict = virsh_sandbox_internal_rest_destroy_sandbox_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestDestroySandboxResponse from a dict
virsh_sandbox_internal_rest_destroy_sandbox_response_from_dict = VirshSandboxInternalRestDestroySandboxResponse.from_dict(virsh_sandbox_internal_rest_destroy_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


