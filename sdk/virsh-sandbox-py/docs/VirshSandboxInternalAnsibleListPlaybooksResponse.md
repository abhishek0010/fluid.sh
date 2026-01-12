# VirshSandboxInternalAnsibleListPlaybooksResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbooks** | [**List[VirshSandboxInternalStorePlaybook]**](VirshSandboxInternalStorePlaybook.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_ansible_list_playbooks_response import VirshSandboxInternalAnsibleListPlaybooksResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalAnsibleListPlaybooksResponse from a JSON string
virsh_sandbox_internal_ansible_list_playbooks_response_instance = VirshSandboxInternalAnsibleListPlaybooksResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalAnsibleListPlaybooksResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_ansible_list_playbooks_response_dict = virsh_sandbox_internal_ansible_list_playbooks_response_instance.to_dict()
# create an instance of VirshSandboxInternalAnsibleListPlaybooksResponse from a dict
virsh_sandbox_internal_ansible_list_playbooks_response_from_dict = VirshSandboxInternalAnsibleListPlaybooksResponse.from_dict(virsh_sandbox_internal_ansible_list_playbooks_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


