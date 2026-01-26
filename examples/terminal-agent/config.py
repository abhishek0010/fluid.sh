"""
Configuration management for the terminal agent.
"""
import json
import os
from pathlib import Path
from typing import Any, Dict

CONFIG_DIR = Path(os.path.expanduser("~/.config/terminal-agent"))
CONFIG_FILE = CONFIG_DIR / "config.json"

DEFAULT_CONFIG = {
    "sandbox_api_base": "http://localhost:8080"
}

def load_config() -> Dict[str, Any]:
    """Load configuration from file."""
    if not CONFIG_FILE.exists():
        return DEFAULT_CONFIG.copy()
    
    try:
        with open(CONFIG_FILE, "r") as f:
            return json.load(f)
    except Exception:
        return DEFAULT_CONFIG.copy()

def save_config(config: Dict[str, Any]) -> None:
    """Save configuration to file."""
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    with open(CONFIG_FILE, "w") as f:
        json.dump(config, f, indent=2)

def get_sandbox_api_base() -> str:
    """Get sandbox API base from config or environment."""
    # Environment variable takes precedence for overrides
    env_val = os.getenv("SANDBOX_API_BASE")
    if env_val:
        return env_val
    
    config = load_config()
    return config.get("sandbox_api_base", DEFAULT_CONFIG["sandbox_api_base"])

def set_sandbox_api_base(url: str) -> None:
    """Set sandbox API base in config."""
    config = load_config()
    config["sandbox_api_base"] = url
    save_config(config)

def ensure_config_exists() -> bool:
    """
    Check if config exists. 
    Returns True if exists (or env var set), False if setup needed.
    """
    if os.getenv("SANDBOX_API_BASE"):
        return True
    return CONFIG_FILE.exists()
