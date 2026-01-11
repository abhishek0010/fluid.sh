# VirshSandboxInternalAnsibleReorderTasksRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task_ids** | **List[str]** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_reorder_tasks_request import VirshSandboxInternalAnsibleReorderTasksRequest

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleReorderTasksRequest from a JSON string
virsh_sandbox_internal_ansible_reorder_tasks_request_instance = VirshSandboxInternalAnsibleReorderTasksRequest.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleReorderTasksRequest.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_reorder_tasks_request_dict = virsh_sandbox_internal_ansible_reorder_tasks_request_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleReorderTasksRequest from a dict
virsh_sandbox_internal_ansible_reorder_tasks_request_from_dict = VirshSandboxInternalAnsibleReorderTasksRequest.from_dict(virsh_sandbox_internal_ansible_reorder_tasks_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


