#!/usr/bin/env python3
"""Post-process generated SDK for better quality"""

import re
from pathlib import Path


def clean_docstrings(file_path: Path):
    """Remove redundant info from docstrings"""
    content = file_path.read_text()

    # Remove "noqa: E501" comments
    content = re.sub(r'\s*# noqa: E501', '', content)

    # Remove excessive blank lines
    content = re.sub(r'\n{3,}', '\n\n', content)

    file_path.write_text(content)


def add_py_typed(sdk_dir: Path):
    """Add py.typed marker for type checkers"""
    (sdk_dir / "py.typed").touch()


def create_service_wrappers(sdk_dir: Path):
    """Create clean service wrapper classes"""

    wrapper_code = '''"""Service wrappers for cleaner API"""

from .configuration import Configuration
from .api_client import ApiClient

class ServiceA:
    """Virsh Sandbox client

    Example:
        >>> service = VirshSandbox(config)
        >>> service.users.list()
    """

    def __init__(self, config: Configuration):
        self._client = ApiClient(config)

        # Import APIs lazily
        from .api.users_api import UsersApi
        from .api.products_api import ProductsApi

        self.users = UsersApi(self._client)
        self.products = ProductsApi(self._client)


class TmuxClient:
    """Tmux Client client"""

    def __init__(self, config: Configuration):
        self._client = ApiClient(config)

        from .api.analytics_api import AnalyticsApi
        from .api.reports_api import ReportsApi

        self.analytics = AnalyticsApi(self._client)
        self.reports = ReportsApi(self._client)
'''

    (sdk_dir / "services.py").write_text(wrapper_code)


def main():
    sdk_dir = Path("virsh-sandbox-python/virsh_sandbox")

    # Clean all Python files
    for py_file in sdk_dir.rglob("*.py"):
        clean_docstrings(py_file)

    # Add type checking support
    add_py_typed(sdk_dir)

    # Create service wrappers
    create_service_wrappers(sdk_dir)

    print("SDK polished!")


if __name__ == "__main__":
    main()
