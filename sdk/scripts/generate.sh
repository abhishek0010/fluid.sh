#!/bin/bash
# scripts/generate.sh

set -e

echo "Generating SDK..."

# Merge OpenAPI specs first
npx openapi-merge-cli --config openapi/openapi-merge.json

# Generate with custom templates
docker run --rm \
  -v ${PWD}:/local \
  openapitools/openapi-generator-cli generate --skip-validate-spec \
  -i /local/openapi/combined.yaml \
  -g python \
  -o /local/virsh-sandbox-python/ \
  -c /local/.openapi-generator/config.yaml \
  -t /local/.openapi-generator/templates/python/

echo "Running polish script..."
python3 scripts/polish_sdk.py

echo "Running tests..."
cd virsh-sandbox-python
pip install -e ".[dev]"
black .
isort .
mypy virsh_sandbox
pytest

echo "Finished !"
