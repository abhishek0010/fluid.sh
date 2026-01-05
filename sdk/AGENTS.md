# SDK Development Guidelines

## Python SDK (`virsh-sandbox-py`)

### Development Scripts

#### Setup & Dependencies

```bash
cd sdk/virsh-sandbox-py

# Install dependencies
pip install -r requirements.txt

# Install dev/test dependencies
pip install -r test-requirements.txt
```

#### Build

```bash
cd sdk/virsh-sandbox-py

# Build the package
python -m build

# Or using setuptools directly
python setup.py sdist bdist_wheel
```

#### Testing

```bash
cd sdk/virsh-sandbox-py

# Run tests
python3 -m pytest test/test_client.py -v

# Run tests with coverage
pytest --cov=virsh_sandbox

# Run tests via tox (multiple Python versions)
tox
```

#### Linting & Type Checking

```bash
cd sdk/virsh-sandbox-py

# Type checking with mypy
mypy virsh_sandbox/client.py

# Full mypy check
mypy virsh_sandbox

# Format code
black .
isort .

# Lint with flake8
flake8 virsh_sandbox
```

#### Code Generation (from OpenAPI spec)

```bash
cd sdk

# Generate SDK from OpenAPI spec (runs full pipeline)
./scripts/generate.sh

# This script:
# 1. Merges OpenAPI specs
# 2. Generates Python client code
# 3. Runs polish script for customizations
# 4. Formats and type-checks the output
# 5. Runs tests
```

#### Publishing

```bash
cd sdk

# Publish to PyPI (requires credentials)
./scripts/publish.sh
```

### Return Type Convention

All SDK client methods MUST return Python dictionaries (`dict`) rather than response objects.

- Response objects from the underlying API are automatically converted using the `_to_dict()` helper function
- Methods that return a single item should return `Dict[str, Any]` or a specific `TypedDict`
- Methods that return a list should return `List[Dict[str, Any]]` or `List[SomeTypedDict]`
- This ensures consistent, JSON-serializable responses that are easy to work with

### Type Annotations

All SDK functions MUST have correct type annotations:

1. **Parameters**: Every parameter must have a type annotation
   ```python
   def create_sandbox(
       self,
       agent_id: Optional[str] = None,
       cpu: Optional[int] = None,
       source_vm_name: Optional[str] = None,
   ) -> CreateSandboxResponseDict:
   ```

2. **Return Types**: Every function must have a return type annotation
   - Use `TypedDict` classes for structured responses (defined in `client.py`)
   - Use `Dict[str, Any]` for generic/dynamic responses
   - Use `List[SomeTypedDict]` for list responses
   - Use `None` for functions that don't return a value

3. **Instance Variables**: Class instance variables should be typed in `__init__`
   ```python
   def __init__(self, api: SandboxApi) -> None:
       self._api: SandboxApi = api
   ```

4. **TypedDict Definitions**: Define TypedDict classes for all response types
   ```python
   class SandboxDict(TypedDict, total=False):
       id: str
       agent_id: str
       state: str
       ip_address: Optional[str]
   ```

### Example

```python
# Correct - returns a typed dict with proper annotations
def create_sandbox(
    self,
    agent_id: Optional[str] = None,
    source_vm_name: Optional[str] = None,
) -> CreateSandboxResponseDict:
    """Create a new sandbox.
    
    Args:
        agent_id: Required agent identity.
        source_vm_name: Required; name of existing VM to clone from.
    
    Returns:
        CreateSandboxResponseDict: Dictionary containing the created sandbox.
    """
    request = InternalRestCreateSandboxRequest(
        agent_id=agent_id,
        source_vm_name=source_vm_name,
    )
    return _to_dict(self._api.create_sandbox(request=request))

# Result is a dict:
# {"sandbox": {"id": "SBX-123", "state": "CREATED", ...}}
```

### Adding New Methods

When adding new methods to the client:

1. Define a `TypedDict` for the response type if it doesn't exist:
   ```python
   class NewResponseDict(TypedDict, total=False):
       field1: str
       field2: int
       optional_field: Optional[str]
   ```

2. Add type annotations to all parameters and return type:
   ```python
   def new_method(
       self,
       required_param: str,
       optional_param: Optional[int] = None,
   ) -> NewResponseDict:
   ```

3. Wrap the API call return value with `_to_dict()`:
   ```python
   return _to_dict(self._api.new_method(param=param))
   ```

4. Add docstrings with Args and Returns sections

5. Add corresponding tests in `test/test_client.py` to verify:
   - Dict return types
   - Correct parameter handling

### Documentation

All functions should have docstrings that include:

1. A brief description of what the function does
2. An `Args:` section documenting each parameter
3. A `Returns:` section documenting the return type

```python
def get_sandbox_session(
    self,
    session_name: str,
) -> SandboxSessionInfoDict:
    """Get sandbox session.

    Args:
        session_name: The session name to retrieve.

    Returns:
        SandboxSessionInfoDict: Dictionary containing session info.
    """
    return _to_dict(self._api.get_sandbox_session(session_name=session_name))
```
