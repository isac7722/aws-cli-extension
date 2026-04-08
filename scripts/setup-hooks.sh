#!/bin/sh
set -e

git config core.hooksPath .githooks
echo "Git hooks configured (.githooks)."

if ! command -v goimports >/dev/null 2>&1; then
  echo "Installing goimports..."
  go install golang.org/x/tools/cmd/goimports@latest
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "Installing golangci-lint..."
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

echo "Done! Go files will be auto-formatted on commit, and lint checked on push."
