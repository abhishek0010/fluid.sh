# VirshSandboxInternalRestSessionEndResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**session_id** | **str** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_session_end_response import VirshSandboxInternalRestSessionEndResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestSessionEndResponse from a JSON string
virsh_sandbox_internal_rest_session_end_response_instance = VirshSandboxInternalRestSessionEndResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestSessionEndResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_session_end_response_dict = virsh_sandbox_internal_rest_session_end_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestSessionEndResponse from a dict
virsh_sandbox_internal_rest_session_end_response_from_dict = VirshSandboxInternalRestSessionEndResponse.from_dict(virsh_sandbox_internal_rest_session_end_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


