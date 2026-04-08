#!/bin/sh
set -e

git config core.hooksPath .githooks
echo "Git hooks configured (.githooks)."

if ! command -v goimports >/dev/null 2>&1; then
  echo "Installing goimports..."
  go install golang.org/x/tools/cmd/goimports@latest
fi

echo "Done! Go files will be auto-formatted on commit."
