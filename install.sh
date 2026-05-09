#!/usr/bin/env bash
# CiscoBuddy installer for Linux/macOS.
# Works in two modes:
#   1) Local: cd into the cloned repo and run ./install.sh
#   2) Remote: curl -fsSL <url-to-this-script> | bash
#      (clones the repo into a temp dir, builds, installs)

set -euo pipefail

REPO_URL="https://github.com/papura-octavian/CiscoBuddy.git"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="ciscobuddy"

# If running locally (script file exists on disk), cd into its directory
SCRIPT_PATH="${BASH_SOURCE[0]:-$0}"
if [ -f "$SCRIPT_PATH" ]; then
    cd "$(dirname "$SCRIPT_PATH")"
fi

# If we don't have source files here, clone the repo into a temp dir
if [ ! -f "main.go" ] || [ ! -f "go.mod" ]; then
    if ! command -v git >/dev/null 2>&1; then
        echo "error: 'git' is not installed." >&2
        exit 1
    fi
    TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "$TMP_DIR"' EXIT
    echo ">> Cloning $REPO_URL ..."
    git clone --depth 1 "$REPO_URL" "$TMP_DIR/repo"
    cd "$TMP_DIR/repo"
fi

if ! command -v go >/dev/null 2>&1; then
    echo "error: 'go' is not installed or not in PATH." >&2
    echo "Install Go from: https://go.dev/dl/" >&2
    exit 1
fi

echo ">> Building $BIN_NAME ..."
go build -o "$BIN_NAME" .

mkdir -p "$INSTALL_DIR"
install -m 0755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"

echo ">> Installed to: $INSTALL_DIR/$BIN_NAME"

case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        echo ">> $INSTALL_DIR is already in PATH."
        echo ">> Done. Run: $BIN_NAME -ip ..."
        ;;
    *)
        echo ""
        echo "WARNING: $INSTALL_DIR is NOT in PATH."
        echo "Add the following line to ~/.bashrc (or ~/.zshrc):"
        echo ""
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
        echo "Then: source ~/.bashrc  (or open a new terminal)"
        ;;
esac
