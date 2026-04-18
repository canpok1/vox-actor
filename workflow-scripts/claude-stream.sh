#!/usr/bin/env bash
set -euo pipefail

claude "$@" --output-format stream-json --verbose --include-partial-messages \
  | jq -rj 'if .type == "text_delta" then .text elif .type == "result" then .result else empty end'

echo
