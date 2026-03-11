#!/bin/bash

# 標準入力からJSON形式でセッション情報を受け取る
input=$(cat)

# ヘルパー関数
get_model_name() { echo "$input" | jq -r '.model.display_name // "Unknown"'; }
get_context_window_size() { echo "$input" | jq -r '.context_window.context_window_size // 0'; }
get_current_usage() { echo "$input" | jq '.context_window.current_usage'; }
get_cost() { echo "$input" | jq -r '.cost.total_cost_usd // empty'; }

# 入力トークン総数（通常の入力 + キャッシュ作成 + キャッシュ読み取り）
get_input_tokens() {
  echo "$input" | jq -r '
    .context_window.current_usage |
    ((.input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.cache_read_input_tokens // 0))
  '
}

# 出力トークン数
get_output_tokens() {
  echo "$input" | jq -r '.context_window.current_usage.output_tokens // 0'
}

# トークン数を読みやすい形式にフォーマット（1000以上は "k" 単位）
format_tokens() {
  local tokens=$1
  if [ "$tokens" -ge 1000 ]; then
    echo "$tokens" | awk '{printf "%.1fk", $1/1000}'
  else
    echo "$tokens"
  fi
}

# コンテキスト使用率を算出
calc_context_percent() {
  local context_size usage
  context_size=$(get_context_window_size)
  usage=$(get_current_usage)

  if [ "$usage" != "null" ] && [[ "$context_size" =~ ^[1-9][0-9]*$ ]]; then
    echo "$usage" | jq -r "((.input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.cache_read_input_tokens // 0)) * 100 / $context_size | floor"
  else
    echo "0"
  fi
}

# 各値を取得
MODEL=$(get_model_name)
CONTEXT_PERCENT=$(calc_context_percent)
INPUT_TOKENS=$(format_tokens "$(get_input_tokens)")
OUTPUT_TOKENS=$(format_tokens "$(get_output_tokens)")
COST=$(get_cost)

# ステータスラインを出力
if [ -n "$COST" ]; then
  printf "Model: %s | Context: %s%% | Tokens: %s in / %s out | Cost: \$%.2f" \
    "$MODEL" "$CONTEXT_PERCENT" "$INPUT_TOKENS" "$OUTPUT_TOKENS" "$COST"
else
  printf "Model: %s | Context: %s%% | Tokens: %s in / %s out" \
    "$MODEL" "$CONTEXT_PERCENT" "$INPUT_TOKENS" "$OUTPUT_TOKENS"
fi
