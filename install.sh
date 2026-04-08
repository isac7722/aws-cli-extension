#!/usr/bin/env bash
# ============================================================
# aws-cli-extension installer
# Usage: curl -fsSL https://raw.githubusercontent.com/isac7722/aws-cli-extension/main/install.sh | bash
#
# Downloads the latest awse binary from GitHub Releases.
# ============================================================

set -e

REPO="isac7722/aws-cli-extension"
INSTALL_DIR="${AWSE_INSTALL_DIR:-/usr/local/bin}"

# ── Utility functions ───────────────────────────────────────

log()     { echo "  $1"; }
success() { echo "✔  $1"; }
warn()    { echo "⚠  $1"; }
error()   { echo "✗  $1"; exit 1; }

# ── Detect OS and Architecture ──────────────────────────────

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin*) os="darwin" ;;
    Linux*)  os="linux" ;;
    *)       error "Unsupported OS: $(uname -s)" ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)             error "Unsupported architecture: $(uname -m)" ;;
  esac

  echo "${os}_${arch}"
}

# ── Detect Shell ────────────────────────────────────────────

detect_shell() {
  if [[ -n "$ZSH_VERSION" ]] || [[ "$SHELL" == */zsh ]]; then
    echo "zsh"
  else
    echo "bash"
  fi
}

detect_rc_file() {
  local shell_type="$1"
  case "$shell_type" in
    zsh)  echo "$HOME/.zshrc" ;;
    bash)
      if [[ -f "$HOME/.bashrc" ]]; then
        echo "$HOME/.bashrc"
      elif [[ -f "$HOME/.bash_profile" ]]; then
        echo "$HOME/.bash_profile"
      else
        echo "$HOME/.bashrc"
      fi
      ;;
    *)    echo "$HOME/.profile" ;;
  esac
}

# ── Start ───────────────────────────────────────────────────

echo ""
echo "╔══════════════════════════════════════╗"
echo "║    aws-cli-extension Installer       ║"
echo "╚══════════════════════════════════════╝"
echo ""

# ── 1. Download binary ──────────────────────────────────────

PLATFORM="$(detect_platform)"
log "Detected platform: $PLATFORM"

# Get latest release tag
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$LATEST_TAG" ]]; then
  error "Failed to determine latest release"
fi
log "Latest version: $LATEST_TAG"

VERSION="${LATEST_TAG#v}"
ARCHIVE_NAME="aws-cli-extension_${VERSION}_${PLATFORM}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

log "Downloading ${ARCHIVE_NAME}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"
tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

# Install binary
if [[ -w "$INSTALL_DIR" ]]; then
  cp "$TMP_DIR/awse" "$INSTALL_DIR/awse"
  chmod +x "$INSTALL_DIR/awse"
else
  log "Requesting sudo to install to $INSTALL_DIR..."
  sudo cp "$TMP_DIR/awse" "$INSTALL_DIR/awse"
  sudo chmod +x "$INSTALL_DIR/awse"
fi
success "Installed awse to $INSTALL_DIR/awse"

# ── 2. Shell integration ────────────────────────────────────

SHELL_TYPE="$(detect_shell)"
RC_FILE="$(detect_rc_file "$SHELL_TYPE")"

MARKER_START="# >>> aws-cli-extension >>>"
MARKER_END="# <<< aws-cli-extension <<<"

if grep -q "$MARKER_START" "$RC_FILE" 2>/dev/null; then
  success "Shell integration already configured in $RC_FILE"
else
  SOURCE_BLOCK="
${MARKER_START}
eval \"\$(command awse init ${SHELL_TYPE})\"
${MARKER_END}"

  touch "$RC_FILE"
  printf '%s\n' "$SOURCE_BLOCK" >> "$RC_FILE"
  success "Shell integration added to $RC_FILE"
fi

# ── Done ────────────────────────────────────────────────────

echo ""
echo "╔══════════════════════════════════════╗"
echo "║       Installation Complete!         ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "  Apply changes immediately:"
echo "    source $RC_FILE"
echo ""
echo "  Usage:"
echo "    awse doctor       # check AWS CLI setup"
echo "    awse user list    # list AWS profiles"
echo "    awse user switch  # switch profile"
echo "    awse ssm          # browse SSM parameters"
echo ""
