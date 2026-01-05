# Agent Example Development Guidelines

This is a Python example demonstrating how to use the virsh-sandbox SDK with an AI agent using OpenAI function calling.

## Prerequisites

- Python 3.12+
- [uv](https://github.com/astral-sh/uv) (recommended) or pip
- A running virsh-sandbox API server at `http://localhost:8080`
- OpenAI API key

## Setup

### Install Dependencies

Using `uv` (recommended):

```bash
uv sync
```

Using pip:

```bash
pip install -e .
```

### Environment Variables

```bash
export OPENAI_API_KEY="your-api-key"
```

## Development Scripts

### Run the Agent

```bash
# Using uv
uv run python main.py

# Using pip
python main.py
```

### Install/Update Dependencies

```bash
# Add a new dependency
uv add <package-name>

# Update all dependencies
uv sync --upgrade

# Lock dependencies
uv lock
```

### Type Checking (Optional)

```bash
# Using mypy
uv run mypy *.py

# Or install and run directly
pip install mypy
mypy *.py
```

### Linting (Optional)

```bash
# Using ruff
uv run ruff check .
uv run ruff format .

# Or using black/isort
uv run black .
uv run isort .
```

## Project Structure

```
agent-example/
├── main.py           # AI agent loop using OpenAI function calling
├── tools.py          # Tool definitions for the LLM
├── configuration.py  # Configuration settings
├── pyproject.toml    # Project dependencies and metadata
└── uv.lock           # Locked dependencies
```

## Key Files

| File | Description |
|------|-------------|
| `main.py` | Main entry point - runs the AI agent loop |
| `tools.py` | Defines tools/functions available to the LLM |
| `configuration.py` | Configuration for API endpoints and settings |
| `pyproject.toml` | Project metadata and dependencies |

## Testing

Currently no automated tests. To test manually:

1. Ensure the virsh-sandbox API is running
2. Set the `OPENAI_API_KEY` environment variable
3. Run `uv run python main.py`
4. Interact with the agent

## Troubleshooting

### Connection Issues

```bash
# Check if API is running
curl http://localhost:8080/v1/health
```

### Missing Dependencies

```bash
# Reinstall all dependencies
rm -rf .venv
uv sync
```

### Python Version Issues

```bash
# Check Python version (requires 3.12+)
python --version

# Use uv to manage Python version
uv python install 3.12
```
