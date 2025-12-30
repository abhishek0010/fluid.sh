"""Configuration for Virsh Sandbox"""

from typing import Optional, Dict
import os


class Configuration:
    """API client configuration

    Example:
        >>> config = Configuration(
        ...     api_key="your-key",
        ...     host="https://api.example.com"
        ... )
    """

    def __init__(
        self,
        host: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        api_key_prefix: Optional[str] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        access_token: Optional[str] = None,
        **kwargs,
    ):
        """Initialize configuration

        Args:
            host: API base URL
            api_key: API key for authentication
            api_key_prefix: Prefix for API key (e.g., 'Bearer')
            username: Username for basic auth
            password: Password for basic auth
            access_token: OAuth access token
        """
        self.host = host

        # Try environment variables first
        self.api_key = api_key or os.getenv("VIRSH_SANDBOX_API_KEY")
        self.api_key_prefix = api_key_prefix or "Bearer"
        self.username = username or os.getenv("VIRSH_SANDBOX_USERNAME")
        self.password = password or os.getenv("VIRSH_SANDBOX_PASSWORD")
        self.access_token = access_token or os.getenv("VIRSH_SANDBOX_ACCESS_TOKEN")

        # Additional settings
        self.verify_ssl = kwargs.get("verify_ssl", True)
        self.ssl_ca_cert = kwargs.get("ssl_ca_cert")
        self.cert_file = kwargs.get("cert_file")
        self.key_file = kwargs.get("key_file")
        self.timeout = kwargs.get("timeout", 60)

    @classmethod
    def from_env(cls) -> "Configuration":
        """Create configuration from environment variables"""
        return cls()
