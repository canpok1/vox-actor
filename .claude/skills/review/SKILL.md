---
name: review
description: 変更コードに対してドキュメント整合性とアーキテクチャ妥当性を検証するレビュースキル。報告のみを行い、コードやドキュメントの修正は一切行わない。実装後のチェックやレビュー観点の確認が必要な場面では積極的に使用する。
context: fork
agent: general-purpose
allowed-tools: Agent, Bash, Read, Grep, Glob
user-invocable: true
---

## 手順

以下の3つのサブエージェントを同時並行で実行する。

- `doc-validator` サブエージェントを呼び出して、ドキュメントの整合性をチェックする
  - `context: fork` + `agent: doc-validator` で呼び出す
- `architecture-reviewer` サブエージェントを呼び出して、レイヤー責務ルール（`docs/development/layer-rules.md`）に基づくアーキテクチャの妥当性をチェックする
  - `context: fork` + `agent: architecture-reviewer` で呼び出す
- `flaky-test-reviewer` サブエージェントを呼び出して、flaky テストになりやすいパターンをチェックする
  - `context: fork` + `agent: flaky-test-reviewer` で呼び出す

全サブエージェントからの指摘を集約し、呼び出し元に報告する。修正が必要な場合は呼び出し元が精査して対応する。
