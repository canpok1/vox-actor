---
name: assign-issues
description: open状態のIssueを優先度順に評価し、上位2件に assign-to-claude ラベルを付与する。
context: fork
agent: issue-assigner
disable-model-invocation: false
user-invocable: true
---

open状態のIssueを確認し、優先度に基づいてラベルを付与してください。
