---
name: review
description: コード品質レビュー、ドキュメント整合性チェック、アーキテクチャ妥当性チェックを統合的に実施するスキル。/simplify でコード品質をレビューし、2つのサブエージェント（doc-validator・architecture-reviewer）でドキュメント最新性とレイヤー責務の妥当性を検証する。
context: fork
agent: general-purpose
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
user-invocable: true
---

自己レビュー（コード品質 + ドキュメント整合性 + アーキテクチャ妥当性）を行います。

## 手順

1. `/simplify` スキルを呼び出して、変更コードの再利用性・品質・効率のレビューと改善を行う
2. 以下の2つを同時並行で実行する
   - `doc-validator` サブエージェントを呼び出して、ドキュメントの整合性をチェックする
     - `context: fork` + `agent: doc-validator` で呼び出す
     - 問題が見つかった場合はドキュメントを修正する
   - `architecture-reviewer` サブエージェントを呼び出して、レイヤー責務ルール（`docs/architecture/layer-rules.md`）に基づくアーキテクチャの妥当性をチェックする
     - `context: fork` + `agent: architecture-reviewer` で呼び出す
     - 指摘のみで自動修正はしない。報告された違反はメインエージェントが内容を精査し、必要に応じて修正する
3. 修正があった場合はコミットする

## 同時並行の根拠

- `/simplify` はコードを修正するため、サブエージェントのレビューと競合しうる。そのため `/simplify` を先に実行し、その後にサブエージェント 2 つを並行で実行する
- `doc-validator` はドキュメントの修正、`architecture-reviewer` はコードの指摘のみ（修正なし）で、対象が異なり競合しない
- `/simplify` のコード修正が先に完了しているため、サブエージェントは修正後のコードを正しくレビューできる
