"""Service wrappers for cleaner API"""

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
