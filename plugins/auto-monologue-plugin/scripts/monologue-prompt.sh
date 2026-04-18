#!/usr/bin/env bash
set -euo pipefail

input=$(cat)
stop_hook_active=$(echo "$input" | jq -r '.stop_hook_active // false')

if [ "$stop_hook_active" = "true" ]; then
  exit 0
fi

jq -n '{decision: "block", reason: "区切りがいいので/vox-actor-plugin:monologueスキルを活用して。"}'
