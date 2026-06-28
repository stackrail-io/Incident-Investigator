#!/usr/bin/env bash
# Start the Incident Investigator MCP server for plugin installs.
# Tries, in order: PATH binary, Docker image, then prints install instructions.
set -euo pipefail

IMAGE="${INCIDENT_INVESTIGATOR_IMAGE:-ghcr.io/stackrail-io/incident-investigator:1.0.0}"
INSTALL_URL="https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.sh"

if command -v incident-investigator >/dev/null 2>&1; then
  exec incident-investigator
fi

if command -v docker >/dev/null 2>&1; then
  exec docker run -i --rm "${IMAGE}"
fi

cat >&2 <<EOF
incident-investigator is not installed.

Install the native binary (recommended):
  curl -fsSL ${INSTALL_URL} | bash

Or install Docker and re-run this plugin, or set INCIDENT_INVESTIGATOR_IMAGE.
EOF
exit 1
