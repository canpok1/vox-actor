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
| `2` | 未解決レビュー/CHANGES_REQUESTEDが原因 | ステップ3aへ進む |
| `3` | その他エラー | AIがstderrのエラー内容を分析・対応を試行 → 解決ならフロー先頭（ステップ1）に戻る → 解決不可ならユーザーに通知して中断 |

### ステップ3a: CodeRabbitのapprove漏れチェック（exit 2の場合）

exit 2の場合、レビューコメント対応の**前に**、CodeRabbitがレビュー完了済みなのにapproveし忘れていないかをチェックする。

#### 情報収集

以下のコマンドでCodeRabbitのレビュー状態を取得する:

```bash
# CodeRabbitの最新レビュー状態を取得
gh pr view --repo <REPO> <PR番号> --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | last'

# CodeRabbitのレビューコメント（サマリー）を取得
gh pr view --repo <REPO> <PR番号> --json comments --jq '[.comments[] | select(.author.login=="coderabbitai")] | last | .body'
```

#### AI判断

取得した情報をもとに、以下の基準で判断する:

- **approve漏れと判断する条件**（以下をすべて満たす場合）:
  - CodeRabbitのレビューが存在するが、状態が `APPROVED` ではない
  - レビューコメントの内容から、指摘事項がない（または全て解決済み）と読み取れる
  - 実質的にレビュー完了しているにもかかわらず、approveアクションだけが行われていない

- **approve漏れではないと判断する条件**（以下のいずれかに該当する場合）:
  - 未解決の指摘事項が残っている
  - レビュー状態が `CHANGES_REQUESTED` で、実際に対応すべき変更要求がある
  - レビューがまだ進行中である

#### アクション

- **approve漏れの場合**: PRに `@coderabbitai approve` コメントを投稿 → フロー先頭（ステップ1）に戻る
- **approve漏れではない場合**: ステップ3bへ進む

### ステップ3b: レビューコメント対応（exit 2の場合）

`/pr-comments` スキルでレビューコメントを取得し対応 → コミット・プッシュ → フロー先頭（ステップ1）に戻る（※ステップ1のwait-coderabbit.shがCHANGES_REQUESTED検知時に `@coderabbitai resolve` を自動投稿する）

## 注意事項

- レビューコメント対応時は `/pr-comments` スキルを使用すること
- コンフリクト解消時はコミットメッセージにIssue番号を含めること
- マージ完了後のクリーンアップ（ブランチ削除等）はこのスキルの責務外（GitHub側の自動削除設定に任せる）
