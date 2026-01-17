"""Tests for the LLM abstraction layer."""

from typing import Any
from unittest.mock import MagicMock, patch

import pytest

from llm import (
    LLMConfig,
    LLMProvider,
    LocalProvider,
    OpenAIProvider,
    OpenRouterProvider,
    create_provider,
)


class MockChatCompletion:
    """Mock chat completion response."""

    def __init__(self, content: str = "Hello", tool_calls: list[Any] | None = None) -> None:
        self.choices = [MagicMock()]
        self.choices[0].message.content = content
        self.choices[0].message.tool_calls = tool_calls


class TestLLMConfig:
    """Tests for LLMConfig dataclass."""

    def test_basic_config(self) -> None:
        """Test basic configuration."""
        config = LLMConfig(api_key="test-key", model="gpt-4o")
        assert config.api_key == "test-key"
        assert config.model == "gpt-4o"
        assert config.base_url is None
        assert config.extra_headers is None

    def test_full_config(self) -> None:
        """Test configuration with all fields."""
        config = LLMConfig(
            api_key="test-key",
            model="gpt-4o",
            base_url="https://custom.api.com",
            extra_headers={"X-Custom": "value"},
        )
        assert config.api_key == "test-key"
        assert config.model == "gpt-4o"
        assert config.base_url == "https://custom.api.com"
        assert config.extra_headers == {"X-Custom": "value"}


class TestOpenAIProvider:
    """Tests for OpenAI provider."""

    @patch("llm.OpenAI")
    def test_initialization(self, mock_openai: MagicMock) -> None:
        """Test provider initialization."""
        config = LLMConfig(api_key="test-key", model="gpt-4o")
        provider = OpenAIProvider(config)

        assert provider.name == "openai"
        assert provider.model == "gpt-4o"
        mock_openai.assert_called_once_with(api_key="test-key", base_url=None)

    @patch("llm.OpenAI")
    def test_initialization_with_base_url(self, mock_openai: MagicMock) -> None:
        """Test provider initialization with custom base URL."""
        config = LLMConfig(
            api_key="test-key",
            model="gpt-4o",
            base_url="https://custom.api.com",
        )
        provider = OpenAIProvider(config)

        mock_openai.assert_called_once_with(
            api_key="test-key",
            base_url="https://custom.api.com",
        )

    @patch("llm.OpenAI")
    def test_chat_completion(self, mock_openai: MagicMock) -> None:
        """Test chat completion call."""
        mock_client = MagicMock()
        mock_openai.return_value = mock_client
        mock_client.chat.completions.create.return_value = MockChatCompletion()

        config = LLMConfig(api_key="test-key", model="gpt-4o")
        provider = OpenAIProvider(config)

        messages = [{"role": "user", "content": "Hello"}]
        tools = [{"type": "function", "function": {"name": "test"}}]

        result = provider.chat_completion(messages, tools, "auto")

        mock_client.chat.completions.create.assert_called_once_with(
            model="gpt-4o",
            messages=messages,
            tools=tools,
            tool_choice="auto",
        )

    @patch("llm.OpenAI")
    def test_chat_completion_no_tools(self, mock_openai: MagicMock) -> None:
        """Test chat completion without tools."""
        mock_client = MagicMock()
        mock_openai.return_value = mock_client
        mock_client.chat.completions.create.return_value = MockChatCompletion()

        config = LLMConfig(api_key="test-key", model="gpt-4o")
        provider = OpenAIProvider(config)

        messages = [{"role": "user", "content": "Hello"}]
        result = provider.chat_completion(messages)

        mock_client.chat.completions.create.assert_called_once_with(
            model="gpt-4o",
            messages=messages,
            tools=None,
            tool_choice=None,
        )


class TestOpenRouterProvider:
    """Tests for OpenRouter provider."""

    @patch("llm.OpenAI")
    def test_initialization(self, mock_openai: MagicMock) -> None:
        """Test provider initialization."""
        config = LLMConfig(api_key="or-key", model="anthropic/claude-sonnet-4")
        provider = OpenRouterProvider(config)

        assert provider.name == "openrouter"
        assert provider.model == "anthropic/claude-sonnet-4"
        mock_openai.assert_called_once_with(
            api_key="or-key",
            base_url="https://openrouter.ai/api/v1",
            default_headers=None,
        )

    @patch("llm.OpenAI")
    def test_initialization_with_site_url(self, mock_openai: MagicMock) -> None:
        """Test provider initialization with site URL."""
        config = LLMConfig(api_key="or-key", model="anthropic/claude-sonnet-4")
        provider = OpenRouterProvider(config, site_url="https://myapp.com")

        mock_openai.assert_called_once_with(
            api_key="or-key",
            base_url="https://openrouter.ai/api/v1",
            default_headers={"HTTP-Referer": "https://myapp.com"},
        )

    @patch("llm.OpenAI")
    def test_initialization_with_custom_base_url(self, mock_openai: MagicMock) -> None:
        """Test provider initialization with custom base URL."""
        config = LLMConfig(
            api_key="or-key",
            model="anthropic/claude-sonnet-4",
            base_url="https://custom.openrouter.ai/v1",
        )
        provider = OpenRouterProvider(config)

        mock_openai.assert_called_once_with(
            api_key="or-key",
            base_url="https://custom.openrouter.ai/v1",
            default_headers=None,
        )

    @patch("llm.OpenAI")
    def test_chat_completion(self, mock_openai: MagicMock) -> None:
        """Test chat completion call."""
        mock_client = MagicMock()
        mock_openai.return_value = mock_client
        mock_client.chat.completions.create.return_value = MockChatCompletion()

        config = LLMConfig(api_key="or-key", model="anthropic/claude-sonnet-4")
        provider = OpenRouterProvider(config)

        messages = [{"role": "user", "content": "Hello"}]
        result = provider.chat_completion(messages)

        mock_client.chat.completions.create.assert_called_once_with(
            model="anthropic/claude-sonnet-4",
            messages=messages,
            tools=None,
            tool_choice=None,
        )


class TestLocalProvider:
    """Tests for local model provider."""

    @patch("llm.OpenAI")
    def test_initialization(self, mock_openai: MagicMock) -> None:
        """Test provider initialization."""
        config = LLMConfig(
            api_key="local",
            model="llama3.2",
            base_url="http://localhost:11434/v1",
        )
        provider = LocalProvider(config)

        assert provider.name == "local"
        assert provider.model == "llama3.2"
        mock_openai.assert_called_once_with(
            api_key="local",
            base_url="http://localhost:11434/v1",
        )

    @patch("llm.OpenAI")
    def test_initialization_no_api_key(self, mock_openai: MagicMock) -> None:
        """Test provider initialization without API key."""
        config = LLMConfig(
            api_key="",
            model="llama3.2",
            base_url="http://localhost:11434/v1",
        )
        provider = LocalProvider(config)

        mock_openai.assert_called_once_with(
            api_key="local",
            base_url="http://localhost:11434/v1",
        )

    @patch("llm.OpenAI")
    def test_chat_completion(self, mock_openai: MagicMock) -> None:
        """Test chat completion call."""
        mock_client = MagicMock()
        mock_openai.return_value = mock_client
        mock_client.chat.completions.create.return_value = MockChatCompletion()

        config = LLMConfig(
            api_key="local",
            model="llama3.2",
            base_url="http://localhost:11434/v1",
        )
        provider = LocalProvider(config)

        messages = [{"role": "user", "content": "Hello"}]
        tools = [{"type": "function", "function": {"name": "test"}}]

        result = provider.chat_completion(messages, tools, "auto")

        mock_client.chat.completions.create.assert_called_once_with(
            model="llama3.2",
            messages=messages,
            tools=tools,
            tool_choice="auto",
        )


class TestCreateProvider:
    """Tests for the create_provider factory function."""

    @patch("llm.OpenAI")
    def test_create_openai_provider(self, mock_openai: MagicMock) -> None:
        """Test creating OpenAI provider."""
        provider = create_provider(
            provider_type="openai",
            api_key="test-key",
            model="gpt-4o",
        )

        assert isinstance(provider, OpenAIProvider)
        assert provider.name == "openai"
        assert provider.model == "gpt-4o"

    @patch("llm.OpenAI")
    def test_create_openrouter_provider(self, mock_openai: MagicMock) -> None:
        """Test creating OpenRouter provider."""
        provider = create_provider(
            provider_type="openrouter",
            api_key="or-key",
            model="anthropic/claude-sonnet-4",
        )

        assert isinstance(provider, OpenRouterProvider)
        assert provider.name == "openrouter"
        assert provider.model == "anthropic/claude-sonnet-4"

    @patch("llm.OpenAI")
    def test_create_openrouter_provider_with_site_url(self, mock_openai: MagicMock) -> None:
        """Test creating OpenRouter provider with site URL."""
        provider = create_provider(
            provider_type="openrouter",
            api_key="or-key",
            model="anthropic/claude-sonnet-4",
            site_url="https://myapp.com",
        )

        assert isinstance(provider, OpenRouterProvider)

    @patch("llm.OpenAI")
    def test_create_local_provider(self, mock_openai: MagicMock) -> None:
        """Test creating local provider."""
        provider = create_provider(
            provider_type="local",
            api_key="local",
            model="llama3.2",
            base_url="http://localhost:11434/v1",
        )

        assert isinstance(provider, LocalProvider)
        assert provider.name == "local"
        assert provider.model == "llama3.2"

    def test_unknown_provider_type(self) -> None:
        """Test error on unknown provider type."""
        with pytest.raises(ValueError) as exc_info:
            create_provider(
                provider_type="unknown",
                api_key="test-key",
                model="test-model",
            )

        assert "Unknown provider type: unknown" in str(exc_info.value)
        assert "openai, openrouter, local" in str(exc_info.value)

    @patch("llm.OpenAI")
    def test_create_openai_with_base_url(self, mock_openai: MagicMock) -> None:
        """Test creating OpenAI provider with custom base URL."""
        provider = create_provider(
            provider_type="openai",
            api_key="test-key",
            model="gpt-4o",
            base_url="https://custom.api.com",
        )

        assert isinstance(provider, OpenAIProvider)


class TestLLMProviderInterface:
    """Tests for LLMProvider abstract interface."""

    @patch("llm.OpenAI")
    def test_provider_has_required_properties(self, mock_openai: MagicMock) -> None:
        """Test that providers implement required interface."""
        config = LLMConfig(api_key="test-key", model="gpt-4o")
        provider = OpenAIProvider(config)

        # Should have name property
        assert hasattr(provider, "name")
        assert isinstance(provider.name, str)

        # Should have model property
        assert hasattr(provider, "model")
        assert isinstance(provider.model, str)

        # Should have chat_completion method
        assert hasattr(provider, "chat_completion")
        assert callable(provider.chat_completion)
