#!/usr/bin/env bash
# Install incident-investigator via Docker and print MCP client configuration.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install-docker.sh | bash
#
# Environment variables:
#   INCIDENT_INVESTIGATOR_IMAGE   Docker image (default: ghcr.io/stackrail-io/incident-investigator:latest)
#   INCIDENT_INVESTIGATOR_TAG     Image tag when using default image (default: 0.1.0)

set -euo pipefail

IMAGE="${INCIDENT_INVESTIGATOR_IMAGE:-ghcr.io/stackrail-io/incident-investigator:0.1.0}"
FALLBACK_IMAGE="${INCIDENT_INVESTIGATOR_FALLBACK_IMAGE:-stackrail/incident-investigator:0.1.0}"

err() {
  echo "install-docker.sh: $*" >&2
  exit 1
}

command -v docker >/dev/null 2>&1 || err "docker is required but not installed"

echo "Pulling ${IMAGE}..."
if ! docker pull "${IMAGE}" 2>/dev/null; then
  echo "Could not pull ${IMAGE}."
  echo "Building local image ${FALLBACK_IMAGE} instead..."
  docker build -t "${FALLBACK_IMAGE}" .
  IMAGE="${FALLBACK_IMAGE}"
fi

echo ""
echo "Docker image ready: ${IMAGE}"
echo ""
echo "Run manually:"
echo "  docker run -i --rm ${IMAGE}"
echo ""
echo "Add to your MCP client config:"
echo ""
cat <<EOF
{
  "mcpServers": {
    "incident-investigator": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "${IMAGE}"]
    }
  }
}
EOF
echo ""
