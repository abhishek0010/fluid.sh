# VirshSandboxInternalAnsibleUpdateTaskResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task** | [**VirshSandboxInternalStorePlaybookTask**](VirshSandboxInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_update_task_response import VirshSandboxInternalAnsibleUpdateTaskResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleUpdateTaskResponse from a JSON string
virsh_sandbox_internal_ansible_update_task_response_instance = VirshSandboxInternalAnsibleUpdateTaskResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleUpdateTaskResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_update_task_response_dict = virsh_sandbox_internal_ansible_update_task_response_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleUpdateTaskResponse from a dict
virsh_sandbox_internal_ansible_update_task_response_from_dict = VirshSandboxInternalAnsibleUpdateTaskResponse.from_dict(virsh_sandbox_internal_ansible_update_task_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


