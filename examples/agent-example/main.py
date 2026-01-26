"""
AI Agent for working on infrastructure tasks using the virsh-sandbox API.

This agent uses OpenAI's function calling to interact with the
virsh-sandbox API through a set of defined tools.
"""
import json
from time import sleep
import sys
from uuid import uuid4
from typing import Any
from virsh_sandbox import VirshSandbox, ApiException
from openai import OpenAI
from dotenv import load_dotenv
from pprint import pprint
from tools import TOOLS
from loader_bar import run_blocking_with_loader

load_dotenv()

# ---------------------------
# Configuration
# ---------------------------

API_BASE = "http://localhost:8080"
MODEL = "gpt-5.2"

openai_client = OpenAI()

client = VirshSandbox(API_BASE)

# ---------------------------
# Tool dispatcher
# ---------------------------


def call_tool(name: str, args: dict[str, Any]) -> dict[str, Any]:
    """
    Maps LLM tool calls to the virsh-sandbox API client.
    """
    try:
        if name == "check_health":
            response = client.health.get_health()
            # Convert Pydantic model to dict for JSON serialization
            return {"status": response.status if hasattr(response, 'status') else "healthy"}

        if name == "list_sandboxes":
            response = client.sandbox.list_sandboxes()
            if(response.sandboxes):
                # Convert list of Pydantic models to list of dicts
                sandboxes = [sb.to_dict() if hasattr(sb, 'to_dict') else sb for sb in response.sandboxes]
                return {"sandboxes": sandboxes}
            else:
                return {"sandboxes": []}

        # if name == "create_sandbox":
        #     response = sandbox_client.create_sandbox(
        #         source_vm_name=args["source_vm_name"],
        #         agent_id=args["agent_id"],
        #         vm_name=args.get("vm_name"),
        #         cpu=args.get("cpu"),
        #         memory_mb=args.get("memory_mb"),
        #     )
        #     return response.sandbox.to_dict()

        # if name == "start_sandbox":
        #     response = sandbox_client.start_sandbox(
        #         sandbox_id=args["sandbox_id"],
        #         wait_for_ip=args.get("wait_for_ip", True),
        #     )
        #     return {"ip_address": response.ip_address}

        # if name == "destroy_sandbox":
        #     client.destroy_sandbox(args["sandbox_id"])
        #     return {"success": True, "message": "Sandbox destroyed"}

        if name == "run_command":
            response = client.sandbox.run_sandbox_command(
                id=args["sandbox_id"],
                command=args["command"],
            )
            # Convert Pydantic model to dict for JSON serialization
            output = response.to_dict() if hasattr(response, 'to_dict') else response
            return {"success": True, "message": "Command executed", "output": output}

        # if name == "create_snapshot":
        #     response = sandbox_client.create_snapshot(
        #         sandbox_id=args["sandbox_id"],
        #         name=args["name"],
        #         external=args.get("external", False),
        #     )
        #     return response.snapshot.to_dict()

        # if name == "diff_snapshots":
        #     response = sandbox_client.diff_snapshots(
        #         sandbox_id=args["sandbox_id"],
        #         from_snapshot=args["from_snapshot"],
        #         to_snapshot=args["to_snapshot"],
        #     )
        #     return response.diff.to_dict()

        # if name == "inject_ssh_key":
        #     sandbox_client.inject_ssh_key(
        #         sandbox_id=args["sandbox_id"],
        #         public_key=args["public_key"],
        #         username=args.get("username"),
        #     )
        #     return {"success": True, "message": "SSH key injected"}

        # if name == "create_ansible_job":
        #     response = sandbox_client.create_ansible_job(
        #         vm_name=args["vm_name"],
        #         playbook=args["playbook"],
        #         check=args.get("check", False),
        #     )
        #     return {"job_id": response.job_id, "ws_url": response.ws_url}

        # if name == "get_ansible_job":
        #     response = sandbox_client.get_ansible_job(args["job_id"])
        #     return response.to_dict()
        if name == "create_playbook":
            response = client.ansible_playbooks.create_playbook(
                name=args["name"],
                hosts=args["hosts"],
                become=args.get("become", False),
            )
            output = response.to_dict() if hasattr(response, 'to_dict') else response
            return {
                "success": True,
                "output": output
            }
        if name == "get_playbook":
            response = client.ansible_playbooks.get_playbook(args["playbook_name"])
            if(response.tasks):
                output = response.to_dict() if hasattr(response, 'to_dict') else response
                return {
                    "success": True,
                    "output": output
                }
        if name == "list_playbooks":
            response = client.ansible_playbooks.list_playbooks()
            if(response.playbooks):
                output = response.to_dict() if hasattr(response, 'to_dict') else response
                return {
                    "success": True,
                    "output": output
                }
        if name == "add_playbook_task":
            response = client.ansible_playbooks.add_playbook_task(
                playbook_name=args["playbook_name"],
                name=args["name"],
                module=args["module"],
                params=args.get("params"),
            )
            output = response.to_dict() if hasattr(response, 'to_dict') else response
            return {
                "success": True,
                "output": output
            }
        if name == "update_playbook_task":
            response = client.ansible_playbooks.update_playbook_task(
                playbook_name=args["playbook_name"],
                task_id=args["task_id"],
                name=args.get("name"),
                module=args.get("module"),
                params=args.get("params"),
            )
            output = response.to_dict() if hasattr(response, 'to_dict') else response
            return {
                "success": True,
                "output": output
            }

        if name == "delete_playbook_task":
            client.ansible_playbooks.delete_playbook_task(
                playbook_name=args["playbook_name"],
                task_id=args["task_id"],
            )
            return {"success": True, "message": "Task deleted"}

        if name == "reorder_playbook_tasks":
            client.ansible_playbooks.reorder_playbook_tasks(
                playbook_name=args["playbook_name"],
                task_ids=args["task_ids"],
            )
            return {"success": True, "message": "Tasks reordered"}

        if name == "exit":
            sys.exit(0)
            return {"success": True, "message": "Agent exited"}

        raise ValueError(f"Unknown tool: {name}")

    except ApiException as e:
        return {
            "error": True,
            "status": e.status,
            "reason": e.reason,
            "body": e.body,
        }


# ---------------------------
# Agent loop
# ---------------------------


def run_agent(user_goal: str, sandbox_id: str | None | Any) -> None:
    """
    Run the agent loop to accomplish the user's goal.

    Args:
        user_goal: The task description from the user
    """
    messages: list[dict[str, Any]] = [
        {
            "role": "system",
            "content": (
                "You are an infrastructure automation agent.\n"
                "- Your goal is to complete the user's task by generating an Ansible playbook that recreates the task on a production machine.\n"
                "- Test your updates by running relevant commands on the sandbox and then building out the playbook. Do not make assumptions on outputs.\n"
                "- You MUST use the Ansible tools to create and manage the playbook.\n"
                "- Do not add an extension to the playbook name like .yml or .yaml\n"
                "- Add any steps to the playbook that are necessary to fully recreate the outcome on the production machine.\n"
                "- You can use other tools to explore the environment and test your changes.\n"
                "- Do NOT assume command output.\n"
                "- No shell pipelines or chained commands.\n"
                "- Always check the health of the API before performing operations.\n"
                "- Track progress and report what you're doing.\n"
                f"- Only make changes to the sandbox with ID: {sandbox_id}"
            ),
        },
        {"role": "user", "content": user_goal},
    ]

    while True:
        response = openai_client.chat.completions.create(
            model=MODEL,
            messages=messages,
            tools=TOOLS,
            tool_choice="auto",
        )

        msg = response.choices[0].message

        # Handle tool calls
        if msg.tool_calls:
            # Add assistant message with tool calls
            messages.append(msg.model_dump())

            for tool_call in msg.tool_calls:
                tool_name = tool_call.function.name
                args = json.loads(tool_call.function.arguments)

                print(f"\n[agent] calling tool: {tool_name}")
                print(f"[agent] args: {json.dumps(args, indent=2)}")

                result = call_tool(tool_name, args)

                print(f"[agent] result: {json.dumps(result, indent=2)}")

                messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tool_call.id,
                        "name": tool_name,
                        "content": json.dumps(result),
                    }
                )

                # Check for errors
                if isinstance(result, dict) and result.get("error"):
                    print(f"[agent] tool error: {result}")

        else:
            # Normal assistant message (no tool calls)
            content = msg.content or ""
            messages.append({"role": "assistant", "content": content})
            print(f"\n[agent] {content}")

            # Heuristic stop condition - agent indicates completion
            if any(
                phrase in content.lower()
                for phrase in ["done", "complete", "completed", "finished", "task complete", "exit"]
            ):
                print("\n[agent] Task completed!")
                return

        sleep(0.2)

def main():
    print("Starting Fluid agent...")
    print("=" * 50)
    prompt = "Install 'cowsay' and run it, create an Ansible playbook to recreate the task."
    print("Prompt: ", prompt)
    sandbox = None
    agent_id = str(uuid4())
    try:
        sandbox = run_blocking_with_loader(client.sandbox.create_sandbox, source_vm_name="test-vm-1", agent_id=agent_id, auto_start=True, wait_for_ip=True, request_timeout=180.0, title="Creating sandbox...").sandbox

        if(sandbox and sandbox.id):
            run_agent(prompt, sandbox.id)
    except Exception as e:
        print(f"Error: {e}")
    finally:
        if(sandbox and sandbox.id):
            print("Cleaning up sandbox...")
            client.sandbox.destroy_sandbox(id=sandbox.id)


# ---------------------------
# Entry point
# ---------------------------

if __name__ == "__main__":
   main()
