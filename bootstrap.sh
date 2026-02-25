#!/usr/bin/env bash
# bootstrap.sh — install dev tools and set up the Dockge dev environment
set -euo pipefail

TASK_VERSION="v3.45.3"

install_task() {
    if command -v task &>/dev/null; then
        echo "task already installed: $(task --version)"
        return
    fi

    echo "Installing go-task ${TASK_VERSION}..."

    if command -v go &>/dev/null; then
        echo "  Using go install..."
        go install "github.com/go-task/task/v3/cmd/task@${TASK_VERSION}"
    else
        echo "  Using install script..."
        mkdir -p ~/.local/bin
        sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            echo "  Add ~/.local/bin to your PATH:"
            echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        fi
    fi

    echo "  Done: $(task --version 2>/dev/null || echo 'installed (restart shell to use)')"
}

install_task

echo ""
echo "Running task setup..."
task setup
