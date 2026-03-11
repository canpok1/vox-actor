FROM mcr.microsoft.com/devcontainers/base:noble

ENV CLAUDE_CONFIG_DIR=/home/vscode/.claude

RUN apt-get update && \
    apt-get install -y --no-install-recommends tmux && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /workspaces && chown vscode:vscode /workspaces

USER vscode

RUN mkdir -p $CLAUDE_CONFIG_DIR && chmod 700 $CLAUDE_CONFIG_DIR

RUN curl -fsSL https://claude.ai/install.sh | bash
