#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_DIR="${AWSE_INSTALL_DIR:-/usr/local/bin}"

echo "==> Building awse..."
cd "$PROJECT_DIR"
go build -o awse ./cmd/ae/

echo "==> Installing to $INSTALL_DIR/awse..."
if [[ -w "$INSTALL_DIR" ]]; then
  cp awse "$INSTALL_DIR/awse"
else
  sudo cp awse "$INSTALL_DIR/awse"
fi
chmod +x "$INSTALL_DIR/awse"
rm awse

# ── Shell integration ────────────────────────────────────

MARKER_START="# >>> aws-cli-extension >>>"

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

SHELL_TYPE="$(detect_shell)"
RC_FILE="$(detect_rc_file "$SHELL_TYPE")"
MARKER_END="# <<< aws-cli-extension <<<"

if grep -q "$MARKER_START" "$RC_FILE" 2>/dev/null; then
  echo "==> Shell integration already configured in $RC_FILE"
else
  SOURCE_BLOCK="
${MARKER_START}
eval \"\$(command awse init ${SHELL_TYPE})\"
${MARKER_END}"

  touch "$RC_FILE"
  printf '%s\n' "$SOURCE_BLOCK" >> "$RC_FILE"
  echo "==> Shell integration added to $RC_FILE"
fi

echo ""
echo "Done! Restart your shell or run:"
echo "  source $RC_FILE"
