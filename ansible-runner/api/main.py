from fastapi import FastAPI, WebSocket, HTTPException
from scalar_fastapi import get_scalar_api_reference
import asyncio
from pydantic import BaseModel
import uuid

app = FastAPI(docs_url=None, redoc_url=None)


class AnsibleJobRequest(BaseModel):
    vm_name: str
    playbook: str
    check: bool = False


ALLOWED_PLAYBOOKS = {
    "ping.yml",
}

INVENTORY_PATH = "/ansible/inventory"

JOBS = {}


@app.get("/healthz")
async def health_check():
    return {"status": "ok"}


@app.get("/docs", include_in_schema=False)
async def scalar_html():
    return get_scalar_api_reference(
        openapi_url=app.openapi_url,
        title=app.title,
    )


@app.post("/jobs")
def create_job(req: AnsibleJobRequest):
    if req.playbook not in ALLOWED_PLAYBOOKS:
        raise HTTPException(400, "Playbook not allowed")

    job_id = str(uuid.uuid4())

    # Store job metadata (in-memory for MVP)
    JOBS[job_id] = {
        "vm_name": req.vm_name,
        "playbook": req.playbook,
        "check": req.check,
        "status": "pending",
    }

    return {"job_id": job_id, "ws_url": f"/ws/jobs/{job_id}"}


ANSIBLE_IMAGE = "ansible-sandbox"


@app.websocket("/ws/jobs/{job_id}")
async def run_job(ws: WebSocket, job_id: str):
    await ws.accept()

    job = JOBS.get(job_id)
    if not job:
        await ws.send_text("Invalid job ID")
        await ws.close()
        return

    vm = job["vm_name"]
    playbook = job["playbook"]
    check = job["check"]

    ansible_cmd = (
        f"ansible-playbook -i {INVENTORY_PATH} playbooks/{playbook} --limit {vm}"
    )

    if check:
        ansible_cmd += " --check"

    await ws.send_text(f"Running: {ansible_cmd}\n")

    docker_cmd = [
        "docker",
        "run",
        "--rm",
        "--network",
        "host",
        "--read-only",
        "--pids-limit",
        "128",
        "--memory",
        "512m",
        "-e",
        f"ANSIBLE_CMD={ansible_cmd}",
        "-v",
        "/ansible:/runner:ro",
        "-v",
        "/var/run/libvirt:/var/run/libvirt",
        ANSIBLE_IMAGE,
    ]

    process = await asyncio.create_subprocess_exec(
        *docker_cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.STDOUT,
    )

    JOBS[job_id]["status"] = "running"

    if process.stdout is not None:
        async for line in process.stdout:
            await ws.send_text(line.decode())

    rc = await process.wait()
    JOBS[job_id]["status"] = "finished"

    await ws.send_text(f"\nJob finished (rc={rc})")
    await ws.close()
