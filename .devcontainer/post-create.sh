#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# setup tmux
ln -sf "$SCRIPT_DIR/.tmux.conf" ~/.tmux.conf

curl -fsSL https://claude.ai/install.sh | bash

# setup frontend dependencies and Playwright (Chromium + system deps)
if [ -d "$REPO_ROOT/frontend" ]; then
    (
        cd "$REPO_ROOT/frontend"
        npm ci
        npx playwright install --with-deps chromium
    )
fi

# install @playwright/cli globally so the `playwright-cli` Claude skill can run
npm install -g @playwright/cli
