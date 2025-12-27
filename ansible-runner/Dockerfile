FROM python:3.11-slim

# -----------------------------
# System dependencies
# -----------------------------
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

# -----------------------------
# Install uv
# -----------------------------
RUN pip install --no-cache-dir uv

WORKDIR /app

# -----------------------------
# Dependencies
# -----------------------------
COPY pyproject.toml uv.lock ./
RUN uv pip install --system --no-cache .

# -----------------------------
# App code
# -----------------------------
COPY api/ ./api/

# -----------------------------
# Non-root user
# -----------------------------
RUN useradd -m apiuser
USER apiuser

EXPOSE 8000

CMD ["uvicorn", "api.main:app", "--host", "0.0.0.0", "--port", "8000"]
