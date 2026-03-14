---
name: fix-pr
description: PR のCI待機・レビュー対応・マージを行うスキル。solve-issue のステップ7〜11を分離したもの。
context: fork
agent: general-purpose
allowed-tools: Skill, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: false
user-invocable: true
argument-hint: "<PR番号>"
---

PR $ARGUMENTS に対して、CI待機・レビュー対応・マージを行います。

## フロー

以下のフローを繰り返す。フロー先頭に戻る場合は必ずステップ1から再開する。

### ステップ1: wait-coderabbit.sh を実行

```bash
./scripts/wait-coderabbit.sh <PR番号>
```

| 終了コード | アクション |
|---|---|
| `0` | ステップ2へ進む |
| `1` | AIがエラー内容を分析し対応を試行。解決不可ならユーザーに通知して中断 |

### ステップ2: fix-pr.sh を実行

```bash
./scripts/fix-pr.sh <PR番号>
```

### ステップ3: 終了コードに基づいて対応

| 終了コード | 意味 | アクション |
|---|---|---|
| `0` | マージ完了 | 完了。ユーザーに結果を報告 |
| `1` | コンフリクト要解消 | AIがコンフリクトを解消 → コミット・プッシュ → フロー先頭（ステップ1）に戻る |
| `2` | 未解決レビューが原因 | `/pr-comments` スキルでレビューコメントを取得し対応 → コミット・プッシュ → フロー先頭（ステップ1）に戻る |
| `3` | その他エラー | AIがstderrのエラー内容を分析・対応を試行 → 解決ならフロー先頭（ステップ1）に戻る → 解決不可ならユーザーに通知して中断 |

## 注意事項

- レビューコメント対応時は `/pr-comments` スキルを使用すること
- コンフリクト解消時はコミットメッセージにIssue番号を含めること
- マージ完了後のクリーンアップ（ブランチ削除等）はこのスキルの責務外（GitHub側の自動削除設定に任せる）
