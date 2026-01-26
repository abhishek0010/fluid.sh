"""
Terminal Agent - Claude Code-esque terminal agent for infrastructure automation.

Entry point for the terminal agent application.
"""

import argparse
import asyncio
import os
import sys
import time
from typing import Any

from dotenv import load_dotenv

import config
from agent import AgentLoop
from llm import create_provider
from middleware import create_default_middleware_chain
from tools import ToolRegistry
from tui import run_tui
from session_manager import SessionState
from mcp_manager import MCPManager
from telemetry import get_telemetry

load_dotenv()


# Default system prompt for the terminal agent
DEFAULT_SYSTEM_PROMPT = """You are an infrastructure automation agent running in a terminal.

Your capabilities:
- Execute commands on sandbox VMs
- Create and manage Ansible playbooks
- Plan and execute infrastructure tasks
- Interact with external services via MCP (e.g., Slack, GitHub)

Guidelines:
- Be concise in your responses
- Explain what you're doing before executing tools
- Report errors clearly
- After executing a successful command with `run_command` that modifies the system state (e.g., installing packages, creating files, changing configurations), you SHOULD add it to your Ansible playbook using `add_task`.
- You MUST request human review using `request_review` before performing any potentially destructive actions or when you have completed a significant part of the task and want approval for the generated playbook.
- You SHOULD perform a `dry_run_playbook` before a `run_playbook` on production systems.
- Use the `task_complete` tool when you have finished all your work for a task.
- If Slack tools are available, you can use them to receive tasks from channels or send progress updates.
- If GitHub tools are available, you can use them to manage repositories, read/write files, and store Ansible playbooks.
"""


def create_tool_registry(session_state: SessionState, sandbox_client: Any = None) -> ToolRegistry:
    """
    Create and configure the tool registry with available tools.

    Args:
        session_state: SessionState instance
        sandbox_client: VirshSandbox client instance (optional)
    """
    registry = ToolRegistry()

    # Phase 4 tools
    try:
        from session_manager import ViewSessionTool, RequestReviewTool, TaskCompletionTool
        registry.register(ViewSessionTool(session_state))
        registry.register(RequestReviewTool(session_state))
        registry.register(TaskCompletionTool(session_state))
    except ImportError:
        pass

    # Phase 2 tools
    try:
        from planning_tools import PlanModeTool
        registry.register(PlanModeTool())
    except ImportError:
        pass

    try:
        from ansible_tools import (
            InitPlaybookTool,
            AddTaskTool,
            ViewPlaybookTool,
            ValidatePlaybookTool,
            DryRunPlaybookTool,
            RunPlaybookTool,
        )
        playbook_manager = session_state.playbook_manager
        registry.register(InitPlaybookTool(playbook_manager))
        registry.register(AddTaskTool(playbook_manager))
        registry.register(ViewPlaybookTool(playbook_manager))
        registry.register(ValidatePlaybookTool(playbook_manager))
        registry.register(DryRunPlaybookTool(playbook_manager))
        registry.register(RunPlaybookTool(playbook_manager))
    except ImportError:
        pass

    if sandbox_client:
        try:
            from sandbox_tools import CreateSandboxTool, RunCommandTool
            registry.register(CreateSandboxTool(sandbox_client, session_state))
            registry.register(RunCommandTool(sandbox_client, session_state))
        except ImportError:
            # sandbox_tools might fail to import if dependencies aren't met
            pass

    return registry


async def discover_mcp_servers(mcp_manager: MCPManager) -> None:
    """Discover and connect to MCP servers based on environment variables."""
    # Slack MCP Integration (Task 3.2)
    if os.getenv("SLACK_BOT_TOKEN") and os.getenv("SLACK_APP_TOKEN"):
        # We use npx to run the slack server. Assumes node is installed.
        await mcp_manager.add_server(
            "slack",
            "npx",
            ["-y", "@modelcontextprotocol/server-slack"],
            env={
                "SLACK_BOT_TOKEN": os.getenv("SLACK_BOT_TOKEN"),
                "SLACK_APP_TOKEN": os.getenv("SLACK_APP_TOKEN"),
                "PATH": os.getenv("PATH", "")
            }
        )

    # GitHub MCP Integration (Task 3.4)
    if os.getenv("GITHUB_PERSONAL_ACCESS_TOKEN"):
        await mcp_manager.add_server(
            "github",
            "npx",
            ["-y", "@modelcontextprotocol/server-github"],
            env={
                "GITHUB_PERSONAL_ACCESS_TOKEN": os.getenv("GITHUB_PERSONAL_ACCESS_TOKEN"),
                "PATH": os.getenv("PATH", "")
            }
        )


def print_response(response: Any) -> None:
    """Print an agent response to stdout."""
    if response.content:
        print(f"\n{response.content}")

    for tool_result in response.tool_results:
        status = "error" if tool_result.error else "ok"
        print(f"[{tool_result.name}] ({status})")


def get_sandbox_api_base() -> str:
    """Get the sandbox API base URL from config or environment."""
    return config.get_sandbox_api_base()


def get_provider_config() -> tuple[str, str, str, str | None, str | None]:
    """
    Get provider configuration from environment variables.

    Environment variables:
        LLM_PROVIDER: Provider type (openai, openrouter, local). Default: openai
        LLM_API_KEY: API key (falls back to OPENAI_API_KEY or OPENROUTER_API_KEY)
        LLM_MODEL: Model identifier. Default: gpt-5.2
        LLM_BASE_URL: Base URL override (required for local provider)
        OPENROUTER_SITE_URL: Site URL for OpenRouter attribution

    Returns:
        Tuple of (provider_type, api_key, model, base_url, site_url)
    """
    provider_type = os.getenv("LLM_PROVIDER", "openai")

    # Get API key based on provider
    api_key = os.getenv("LLM_API_KEY")
    if not api_key:
        if provider_type == "openrouter":
            api_key = os.getenv("OPENROUTER_API_KEY")
        elif provider_type == "openai":
            api_key = os.getenv("OPENAI_API_KEY")
        elif provider_type == "local":
            api_key = "local"  # Local models often don't need a key

    # Get model based on provider defaults
    default_models = {
        "openai": "gpt-5.2",
        "openrouter": "anthropic/claude-sonnet-4.5",
        "local": "llama3.2",
    }
    model = os.getenv("LLM_MODEL", default_models.get(provider_type, "gpt-5.2"))

    base_url = os.getenv("LLM_BASE_URL")
    site_url = os.getenv("OPENROUTER_SITE_URL")

    return provider_type, api_key or "", model, base_url, site_url

async def create_agent() -> tuple[AgentLoop, str, str, MCPManager]:
    """
    Create and configure the agent with provider and MCP servers.

    Returns:
        Tuple of (agent, provider_type, model, mcp_manager)
    """
    provider_type, api_key, model, base_url, site_url = get_provider_config()

    if not api_key and provider_type != "local":
        print(f"Error: API key not set for provider '{provider_type}'")
        print("Set LLM_API_KEY or the provider-specific key (OPENAI_API_KEY, OPENROUTER_API_KEY)")
        sys.exit(1)

    if provider_type == "local" and not base_url:
        print("Error: LLM_BASE_URL required for local provider")
        print("Example: LLM_BASE_URL=http://localhost:11434/v1 for Ollama")
        sys.exit(1)

    provider = create_provider(
        provider_type=provider_type,
        api_key=api_key,
        model=model,
        base_url=base_url,
        site_url=site_url,
    )

    # Initialize sandbox client
    sandbox_client = None
    try:
        from virsh_sandbox import VirshSandbox
        api_base = get_sandbox_api_base()
        sandbox_client = VirshSandbox(api_base)
    except ImportError:
        # If virsh-sandbox is not installed, tools won't be available
        pass

    session_state = SessionState()
    registry = create_tool_registry(session_state, sandbox_client)

    # Initialize and discover MCP servers
    mcp_manager = MCPManager()
    await discover_mcp_servers(mcp_manager)
    mcp_manager.register_tools(registry)

    # Configure middleware
    playbook_nudge_enabled = os.getenv("PLAYBOOK_NUDGE_ENABLED", "true").lower() == "true"
    playbook_nudge_max = int(os.getenv("PLAYBOOK_NUDGE_MAX", "3"))
    middleware = create_default_middleware_chain(
        playbook_nudge_enabled=playbook_nudge_enabled,
        playbook_nudge_max=playbook_nudge_max,
    )

    agent = AgentLoop.from_registry(
        provider=provider,
        system_prompt=DEFAULT_SYSTEM_PROMPT,
        registry=registry,
        middleware=middleware,
    )

    return agent, provider_type, model, mcp_manager


async def run_basic_repl(agent: AgentLoop, provider_type: str, model: str) -> None:
    """Run the agent in basic REPL mode without TUI."""
    print(f"Terminal Agent ({provider_type}: {model})")
    print("Type 'exit' or 'quit' to exit, 'reset' to clear history")
    print("Type '/settings' to change configuration")
    print("-" * 40)

    while True:
        try:
            # Note: input() is blocking, but in this basic mode it's acceptable
            # or we could use aioconsole.ainput if we wanted true async input
            user_input = input("\n> ").strip()
        except (KeyboardInterrupt, EOFError):
            print("\nExiting...")
            break

        if not user_input:
            continue

        if user_input.lower() in ("exit", "quit"):
            print("Goodbye!")
            break

        if user_input.lower() == "reset":
            agent.reset()
            print("Conversation reset.")
            continue

        if user_input.lower() == "/settings":
            current_host = config.get_sandbox_api_base()
            print(f"\nCurrent Sandbox Host: {current_host}")
            new_host = input("New Host (leave empty to keep): ").strip()
            if new_host:
                config.set_sandbox_api_base(new_host)
                print("Settings saved. Please restart the agent to apply changes.")
            continue

        responses = await agent.run(user_input)
        for response in responses:
            print_response(response)


async def run_interactive(use_tui: bool = True) -> None:
    """
    Run the agent in interactive mode.

    Args:
        use_tui: Use the textual TUI interface. If False, uses basic REPL.
    """
    start_time = time.time()
    agent, provider_type, model, mcp_manager = await create_agent()

    # Track session start
    telemetry = get_telemetry()
    telemetry.track_session_start(provider_type=provider_type, model=model)

    try:
        if use_tui:
            await run_tui(agent, provider_type, model)
        else:
            await run_basic_repl(agent, provider_type, model)
    finally:
        # Track session end
        duration = time.time() - start_time
        # Get session stats if we can access them
        commands_executed = 0
        sandboxes_created = 0
        telemetry.track_session_end(
            duration_seconds=duration,
            commands_executed=commands_executed,
            sandboxes_created=sandboxes_created,
        )
        telemetry.shutdown()

        await mcp_manager.disconnect_all()


async def run_setup_wizard() -> None:
    """Run interactive setup wizard."""
    print("\n" + "=" * 40)
    print("Terminal Agent Setup")
    print("=" * 40)
    print("It looks like this is your first time running the agent.")
    print("Please configure the Virsh Sandbox host.")

    default_host = "http://localhost:8080"
    host = input(f"\nSandbox Host [{default_host}]: ").strip()

    if not host:
        host = default_host

    config.set_sandbox_api_base(host)
    print(f"\nConfiguration saved to {config.CONFIG_FILE}")
    print("=" * 40 + "\n")


async def main_async() -> None:
    """Async main entry point."""
    parser = argparse.ArgumentParser(
        description="Terminal Agent - infrastructure automation assistant"
    )
    parser.add_argument(
        "--basic",
        action="store_true",
        help="Use basic REPL mode instead of TUI",
    )
    args = parser.parse_args()

    # Check for config
    if not config.ensure_config_exists():
        await run_setup_wizard()

    await run_interactive(use_tui=not args.basic)


def main() -> None:
    """Main entry point."""
    try:
        asyncio.run(main_async())
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
