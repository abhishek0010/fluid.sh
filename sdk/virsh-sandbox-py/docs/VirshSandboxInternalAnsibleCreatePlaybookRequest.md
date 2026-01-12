# VirshSandboxInternalAnsibleCreatePlaybookRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**become** | **bool** |  | [optional] 
**hosts** | **str** |  | [optional] 
**name** | **str** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_create_playbook_request import VirshSandboxInternalAnsibleCreatePlaybookRequest

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleCreatePlaybookRequest from a JSON string
virsh_sandbox_internal_ansible_create_playbook_request_instance = VirshSandboxInternalAnsibleCreatePlaybookRequest.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleCreatePlaybookRequest.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_create_playbook_request_dict = virsh_sandbox_internal_ansible_create_playbook_request_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleCreatePlaybookRequest from a dict
virsh_sandbox_internal_ansible_create_playbook_request_from_dict = VirshSandboxInternalAnsibleCreatePlaybookRequest.from_dict(virsh_sandbox_internal_ansible_create_playbook_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


