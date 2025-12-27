FROM python:3.11-slim

# -----------------------------
# System deps
# -----------------------------
RUN apt-get update && apt-get install -y \
    openssh-client \
    sshpass \
    libvirt-clients \
    && rm -rf /var/lib/apt/lists/*

# -----------------------------
# Python deps
# -----------------------------
RUN pip install --no-cache-dir \
    ansible \
    ansible-core \
    community.libvirt

WORKDIR /runner

# -----------------------------
# Non-root execution
# -----------------------------
RUN useradd -m ansible
USER ansible

# -----------------------------
# Runtime config
# -----------------------------
ENV ANSIBLE_CMD="ansible-playbook --version"

# Shell form ENTRYPOINT allows env expansion
ENTRYPOINT ["sh", "-c", "$ANSIBLE_CMD"]
