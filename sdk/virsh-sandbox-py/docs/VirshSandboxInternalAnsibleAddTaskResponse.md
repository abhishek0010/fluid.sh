# VirshSandboxInternalAnsibleAddTaskResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task** | [**VirshSandboxInternalStorePlaybookTask**](VirshSandboxInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_add_task_response import VirshSandboxInternalAnsibleAddTaskResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleAddTaskResponse from a JSON string
virsh_sandbox_internal_ansible_add_task_response_instance = VirshSandboxInternalAnsibleAddTaskResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleAddTaskResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_add_task_response_dict = virsh_sandbox_internal_ansible_add_task_response_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleAddTaskResponse from a dict
virsh_sandbox_internal_ansible_add_task_response_from_dict = VirshSandboxInternalAnsibleAddTaskResponse.from_dict(virsh_sandbox_internal_ansible_add_task_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


