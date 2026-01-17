"""
LLM abstraction layer for provider-agnostic model access.

Supports OpenAI, OpenRouter, and local models via OpenAI-compatible APIs.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Any

from openai import OpenAI
from openai.types.chat import ChatCompletion


@dataclass
class LLMConfig:
    """Configuration for an LLM provider."""

    api_key: str
    model: str
    base_url: str | None = None
    extra_headers: dict[str, str] | None = None


class LLMProvider(ABC):
    """Abstract base class for LLM providers."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Provider name for identification."""
        ...

    @abstractmethod
    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> ChatCompletion:
        """
        Send a chat completion request.

        Args:
            messages: Conversation messages
            tools: Optional tool definitions in OpenAI format
            tool_choice: Optional tool choice mode ("auto", "none", or specific)

        Returns:
            ChatCompletion response
        """
        ...

    @property
    @abstractmethod
    def model(self) -> str:
        """Current model identifier."""
        ...


class OpenAIProvider(LLMProvider):
    """OpenAI direct API provider."""

    def __init__(self, config: LLMConfig) -> None:
        """
        Initialize OpenAI provider.

        Args:
            config: LLM configuration with api_key and model
        """
        self._config = config
        self._client = OpenAI(
            api_key=config.api_key,
            base_url=config.base_url,
        )

    @property
    def name(self) -> str:
        return "openai"

    @property
    def model(self) -> str:
        return self._config.model

    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> ChatCompletion:
        return self._client.chat.completions.create(
            model=self._config.model,
            messages=messages,
            tools=tools if tools else None,
            tool_choice=tool_choice if tools else None,
        )


class OpenRouterProvider(LLMProvider):
    """OpenRouter API provider for privacy-focused model access."""

    OPENROUTER_BASE_URL = "https://openrouter.ai/api/v1"

    def __init__(self, config: LLMConfig, site_url: str | None = None) -> None:
        """
        Initialize OpenRouter provider.

        Args:
            config: LLM configuration with api_key and model
            site_url: Optional site URL for OpenRouter attribution
        """
        self._config = config
        headers = {"HTTP-Referer": site_url} if site_url else {}
        if config.extra_headers:
            headers.update(config.extra_headers)

        self._client = OpenAI(
            api_key=config.api_key,
            base_url=config.base_url or self.OPENROUTER_BASE_URL,
            default_headers=headers if headers else None,
        )

    @property
    def name(self) -> str:
        return "openrouter"

    @property
    def model(self) -> str:
        return self._config.model

    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> ChatCompletion:
        return self._client.chat.completions.create(
            model=self._config.model,
            messages=messages,
            tools=tools if tools else None,
            tool_choice=tool_choice if tools else None,
        )


class LocalProvider(LLMProvider):
    """Local model provider via OpenAI-compatible API (Ollama, LMStudio, etc)."""

    def __init__(self, config: LLMConfig) -> None:
        """
        Initialize local model provider.

        Args:
            config: LLM configuration with model and base_url
                   api_key can be "ollama" or any placeholder for local servers
        """
        self._config = config
        self._client = OpenAI(
            api_key=config.api_key or "local",
            base_url=config.base_url,
        )

    @property
    def name(self) -> str:
        return "local"

    @property
    def model(self) -> str:
        return self._config.model

    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> ChatCompletion:
        return self._client.chat.completions.create(
            model=self._config.model,
            messages=messages,
            tools=tools if tools else None,
            tool_choice=tool_choice if tools else None,
        )


def create_provider(
    provider_type: str,
    api_key: str,
    model: str,
    base_url: str | None = None,
    site_url: str | None = None,
) -> LLMProvider:
    """
    Factory function to create an LLM provider.

    Args:
        provider_type: One of "openai", "openrouter", or "local"
        api_key: API key for the provider
        model: Model identifier
        base_url: Optional base URL override
        site_url: Optional site URL for OpenRouter attribution

    Returns:
        Configured LLMProvider instance

    Raises:
        ValueError: If provider_type is not recognized
    """
    config = LLMConfig(api_key=api_key, model=model, base_url=base_url)

    if provider_type == "openai":
        return OpenAIProvider(config)
    elif provider_type == "openrouter":
        return OpenRouterProvider(config, site_url=site_url)
    elif provider_type == "local":
        return LocalProvider(config)
    else:
        raise ValueError(
            f"Unknown provider type: {provider_type}. "
            f"Must be one of: openai, openrouter, local"
        )
