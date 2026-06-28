#!/usr/bin/env bash
# Install incident-investigator on macOS or Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.sh | bash
#
# Environment variables:
#   INCIDENT_INVESTIGATOR_VERSION   Version to install (default: latest)
#   INCIDENT_INVESTIGATOR_REPO      GitHub repo (default: stackrail-io/Incident-Investigator)
#   INSTALL_DIR                     Install directory (default: /usr/local/bin, or ~/.local/bin without sudo)

set -euo pipefail

REPO="${INCIDENT_INVESTIGATOR_REPO:-stackrail-io/Incident-Investigator}"
BINARY="incident-investigator"
VERSION="${INCIDENT_INVESTIGATOR_VERSION:-latest}"

err() {
  echo "install.sh: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) err "unsupported OS: $(uname -s) (use Docker or build from source)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) err "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    need_cmd curl
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\?\([^"]*\)".*/\1/p' \
      | head -1)"
    [ -n "$VERSION" ] || err "could not resolve latest release version"
  fi
  # Strip leading v if provided.
  VERSION="${VERSION#v}"
}

pick_install_dir() {
  if [ -n "${INSTALL_DIR:-}" ]; then
    echo "$INSTALL_DIR"
    return
  fi
  if [ -w /usr/local/bin ] 2>/dev/null; then
    echo "/usr/local/bin"
  elif [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
    echo "$HOME/.local/bin"
  else
    err "could not find a writable install directory; set INSTALL_DIR"
  fi
}

main() {
  need_cmd tar
  need_cmd curl

  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  resolve_version

  INSTALL_DIR="$(pick_install_dir)"
  ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"
  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR"' EXIT

  echo "Installing ${BINARY} v${VERSION} for ${OS}/${ARCH}..."
  echo "Downloading ${URL}"
  curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"
  tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"
  BIN_SRC="${TMPDIR}/${BINARY}"
  [ -f "$BIN_SRC" ] || err "archive did not contain ${BINARY}"

  if [ ! -w "$INSTALL_DIR" ]; then
    need_cmd sudo
    sudo install -m 755 "$BIN_SRC" "${INSTALL_DIR}/${BINARY}"
  else
    install -m 755 "$BIN_SRC" "${INSTALL_DIR}/${BINARY}"
  fi

  echo ""
  echo "Installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"
  "${INSTALL_DIR}/${BINARY}" version
  echo ""
  echo "Add to your MCP client config (Cursor, Claude Code, etc.):"
  echo ""
  cat <<EOF
{
  "mcpServers": {
    "incident-investigator": {
      "command": "${INSTALL_DIR}/${BINARY}"
    }
  }
}
EOF
  echo ""
  if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "Note: ${INSTALL_DIR} is not on your PATH. Add it to your shell profile or use the full path above."
  fi
}

main "$@"
