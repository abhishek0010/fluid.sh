# VirshSandboxInternalAnsibleUpdateTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **Dict[str, object]** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_update_task_request import VirshSandboxInternalAnsibleUpdateTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleUpdateTaskRequest from a JSON string
virsh_sandbox_internal_ansible_update_task_request_instance = VirshSandboxInternalAnsibleUpdateTaskRequest.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleUpdateTaskRequest.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_update_task_request_dict = virsh_sandbox_internal_ansible_update_task_request_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleUpdateTaskRequest from a dict
virsh_sandbox_internal_ansible_update_task_request_from_dict = VirshSandboxInternalAnsibleUpdateTaskRequest.from_dict(virsh_sandbox_internal_ansible_update_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


