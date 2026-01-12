# Request Timeouts

When creating sandboxes with `wait_for_ip=True`, the request may take longer than default HTTP timeouts allow. This document explains how to configure timeouts on both the server and client side.

## Server Configuration

The virsh-sandbox server's HTTP write timeout is automatically calculated based on the IP discovery timeout:

```bash
# IP discovery timeout (default: 120 seconds)
export IP_DISCOVERY_TIMEOUT_SEC=120

# HTTP write timeout = IP_DISCOVERY_TIMEOUT_SEC + 30 seconds
# With default settings: 150 seconds
```

For slower VM boot times (e.g., cloud-init heavy images), increase the IP discovery timeout:

```bash
# 5 minutes for IP discovery, 5.5 minutes HTTP timeout
export IP_DISCOVERY_TIMEOUT_SEC=300
```

## SDK Configuration

The Python SDK accepts a `request_timeout` parameter on methods that may take a long time:

### create_sandbox

```python
from virsh_sandbox import Client

client = Client(host="http://localhost:8080")

# Single timeout value (total request timeout)
sandbox = client.sandbox.create_sandbox(
    source_vm_name="base-vm",
    wait_for_ip=True,
    request_timeout=180.0,  # 180 seconds
)

# Tuple for (connect_timeout, read_timeout)
sandbox = client.sandbox.create_sandbox(
    source_vm_name="base-vm",
    wait_for_ip=True,
    request_timeout=(5.0, 180.0),  # 5s connect, 180s read
)
```

### start_sandbox

```python
# Start an existing sandbox and wait for IP
result = client.sandbox.start_sandbox(
    id=sandbox_id,
    wait_for_ip=True,
    request_timeout=180.0,
)
```

### run_command

```python
# Long-running commands may need extended timeouts
result = client.sandbox.run_command(
    id=sandbox_id,
    command="apt-get update && apt-get upgrade -y",
    timeout_sec=600,        # Server-side command timeout
    request_timeout=660.0,  # Client HTTP timeout (command timeout + buffer)
)
```

## Recommended Values

| Operation | Recommended `request_timeout` |
|-----------|------------------------------|
| `create_sandbox` with `wait_for_ip=False` | 30s (default) |
| `create_sandbox` with `wait_for_ip=True` | 180s |
| `start_sandbox` with `wait_for_ip=True` | 180s |
| `run_command` | `timeout_sec` + 60s buffer |

## Alternative: Async IP Discovery

Instead of blocking on `wait_for_ip=True`, you can poll for the IP address:

```python
import time

# Create without waiting
sandbox = client.sandbox.create_sandbox(
    source_vm_name="base-vm",
    auto_start=True,
    wait_for_ip=False,
)

sandbox_id = sandbox.sandbox.id

# Poll for IP
while True:
    result = client.sandbox.discover_ip(sandbox_id)
    if result.ip_address:
        print(f"IP: {result.ip_address}")
        break
    time.sleep(5)
```

This approach avoids long HTTP request timeouts and provides better visibility into the sandbox startup process.
