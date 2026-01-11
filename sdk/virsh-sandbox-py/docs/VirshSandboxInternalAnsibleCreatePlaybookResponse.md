# VirshSandboxInternalAnsibleCreatePlaybookResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbook** | [**VirshSandboxInternalStorePlaybook**](VirshSandboxInternalStorePlaybook.md) |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_create_playbook_response import VirshSandboxInternalAnsibleCreatePlaybookResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleCreatePlaybookResponse from a JSON string
virsh_sandbox_internal_ansible_create_playbook_response_instance = VirshSandboxInternalAnsibleCreatePlaybookResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleCreatePlaybookResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_create_playbook_response_dict = virsh_sandbox_internal_ansible_create_playbook_response_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleCreatePlaybookResponse from a dict
virsh_sandbox_internal_ansible_create_playbook_response_from_dict = VirshSandboxInternalAnsibleCreatePlaybookResponse.from_dict(virsh_sandbox_internal_ansible_create_playbook_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


