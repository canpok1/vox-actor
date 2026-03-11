#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# setup tmux
ln -sf "$SCRIPT_DIR/.tmux.conf" ~/.tmux.conf

# setup .env if not exists
if [ ! -f "$SCRIPT_DIR/.env" ] && [ -f "$SCRIPT_DIR/.env-template" ]; then
    cp "$SCRIPT_DIR/.env-template" "$SCRIPT_DIR/.env"
    echo ".env file created from .env-template"
fi
