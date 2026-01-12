# VirshSandboxInternalAnsibleGetPlaybookResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbook** | [**VirshSandboxInternalStorePlaybook**](VirshSandboxInternalStorePlaybook.md) |  | [optional] 
**tasks** | [**List[VirshSandboxInternalStorePlaybookTask]**](VirshSandboxInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_get_playbook_response import VirshSandboxInternalAnsibleGetPlaybookResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleGetPlaybookResponse from a JSON string
virsh_sandbox_internal_ansible_get_playbook_response_instance = VirshSandboxInternalAnsibleGetPlaybookResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleGetPlaybookResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_get_playbook_response_dict = virsh_sandbox_internal_ansible_get_playbook_response_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleGetPlaybookResponse from a dict
virsh_sandbox_internal_ansible_get_playbook_response_from_dict = VirshSandboxInternalAnsibleGetPlaybookResponse.from_dict(virsh_sandbox_internal_ansible_get_playbook_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


