# VirshSandboxInternalRestRevokeCertificateResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | [optional] 
**message** | **str** |  | [optional] 

## Example

```python
from virsh_sandbox.models.virsh_sandbox_internal_rest_revoke_certificate_response import VirshSandboxInternalRestRevokeCertificateResponse

# TODO update the JSON string below
json = "{}"
# create an instance of VirshSandboxInternalRestRevokeCertificateResponse from a JSON string
virsh_sandbox_internal_rest_revoke_certificate_response_instance = VirshSandboxInternalRestRevokeCertificateResponse.from_json(json)
# print the JSON string representation of the object
print(VirshSandboxInternalRestRevokeCertificateResponse.to_json())

# convert the object into a dict
virsh_sandbox_internal_rest_revoke_certificate_response_dict = virsh_sandbox_internal_rest_revoke_certificate_response_instance.to_dict()
# create an instance of VirshSandboxInternalRestRevokeCertificateResponse from a dict
virsh_sandbox_internal_rest_revoke_certificate_response_from_dict = VirshSandboxInternalRestRevokeCertificateResponse.from_dict(virsh_sandbox_internal_rest_revoke_certificate_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


