# VirshSandboxInternalRestListSandboxesResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sandboxes** | [**List[VirshSandboxInternalRestSandboxInfo]**](VirshSandboxInternalRestSandboxInfo.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_list_sandboxes_response import VirshSandboxInternalRestListSandboxesResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestListSandboxesResponse from a JSON string
virsh_sandbox_internal_rest_list_sandboxes_response_instance = VirshSandboxInternalRestListSandboxesResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestListSandboxesResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_list_sandboxes_response_dict = virsh_sandbox_internal_rest_list_sandboxes_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestListSandboxesResponse from a dict
virsh_sandbox_internal_rest_list_sandboxes_response_from_dict = VirshSandboxInternalRestListSandboxesResponse.from_dict(virsh_sandbox_internal_rest_list_sandboxes_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


