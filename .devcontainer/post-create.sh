#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# setup tmux
ln -sf "$SCRIPT_DIR/.tmux.conf" ~/.tmux.conf
