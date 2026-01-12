# VirshSandboxInternalAnsibleAddTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **object** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_add_task_request import VirshSandboxInternalAnsibleAddTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleAddTaskRequest from a JSON string
virsh_sandbox_internal_ansible_add_task_request_instance = VirshSandboxInternalAnsibleAddTaskRequest.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleAddTaskRequest.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_add_task_request_dict = virsh_sandbox_internal_ansible_add_task_request_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleAddTaskRequest from a dict
virsh_sandbox_internal_ansible_add_task_request_from_dict = VirshSandboxInternalAnsibleAddTaskRequest.from_dict(virsh_sandbox_internal_ansible_add_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


