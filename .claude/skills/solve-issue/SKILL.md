---
name: solve-issue
description: ユーザーが `/solve-issue` で手動実行した場合のみ使用。GitHub Issueの対応を行うスキル。実装、自己レビュー、PR作成、マージ、振り返りまで一連の流れを一気に行う。
context: fork
agent: general-purpose
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: true
user-invocable: true
argument-hint: "[issue-number]"
---

GitHub Issue $ARGUMENTS を対応します。

## 作業メモ

各ステップの実施時に、作業メモをMarkdownファイルへ記録する。

### メモファイルの仕様

- **配置先**: `.claude/memo/`（存在しなければ `mkdir -p` で作成）
- **ファイル名**: 現在のブランチ名をサニタイズ + `.md`
  - サニタイズ: `/:*?"<>|\` をハイフン `-` に変換
  - 取得: `BRANCH=$(git rev-parse --abbrev-ref HEAD); MEMO_FILE=".claude/memo/$(echo "$BRANCH" | tr '/:*?"<>|\\' '-').md"`
- **既存ファイル**: 上書きせず、Readで読み取って追記（既存内容を保持）
- **書き込み方法**: Readツールで現在の内容を読み取り、Writeツールで更新

### メモファイルのテンプレート

```markdown
# Issue #{番号}: {Issueタイトル}

## 目的

{Issueの内容から要約した目的}

## 作業内容

- [ ] {タスク1}
- [ ] {タスク2}

## 作業ログ

### ステップ1: Issue理解 (YYYY-MM-DD HH:MM)
- {内容}
```

### 書き込みルール

- ステップ1でメモファイルを新規作成（目的・作業内容チェックリストを設定）。既存ファイルがある場合は上書きせず追記する
- 各ステップ完了時に作業ログセクションへ追記
- 作業内容チェックリストは完了時にチェックを付ける
- 目的は理解が深まった場合に更新可

### 各ステップの記録粒度

| ステップ | メモ操作 | 記録内容 |
|---|---|---|
| 1. Issue理解 | **新規作成** | 目的の設定、作業内容チェックリストの初期作成（既存ファイルがあれば追記） |
| 2. TDD実装 | 追記 | 実装内容、作成/変更ファイル、遭遇した問題 |
| 3. 自己レビュー | 追記 | 指摘事項と修正内容 |
| 4. lint/format | 追記 | 指摘の有無と修正内容 |
| 5. 重複チェック | 条件付き | 既存PRが見つかった場合のみ記録 |
| 6. PR作成 | 追記 | PR番号とURL |
| 7. fix-pr | 追記 | CI待機・レビュー対応・マージの結果 |
| 8. クリーンアップ | なし | 定型作業のため省略 |
| 9. 振り返り | 追記 | 作成したIssue番号（あれば） |

1. Issue の内容を理解する
  - メモファイルを作成する（既存ファイルがあれば追記）
2. `/tdd` スキルで実装する
3. `/review` スキルで自己レビュー（コード品質 + ドキュメント整合性チェック）を行う
4. lint/formatチェックを実行する（PR作成前の最終ガード）
  - `gofmt -l .` → 出力があれば `gofmt -w .` で修正
  - `golangci-lint run` → 指摘があれば修正
  - 修正した場合はコミットする
5. 同一Issueに対する既存PRの重複チェックを行う
  - コマンド: `gh pr list --repo {owner}/{repo} --search "issue-{番号}" --state all`
  - 検索結果に複数のPRが含まれる場合、以下の優先順位で判断する:
    1. **merged状態のPRが存在する場合**: 既に対応済みのため、処理をスキップして完了する（ステップ12の振り返りのみ実施する）
    2. **open状態のPRが存在する場合**: 新しいPRを作成せず、既存PRに対してステップ7（fix-pr）を継続する
       - 複数のopen PRがある場合は、最新のものを対象とする
       - PR番号の取得例: `gh pr list --repo {owner}/{repo} --search "issue-{番号}" --state open --json number --jq '.[0].number'`
    3. **closed状態（マージされずにクローズ）のPRのみ存在する場合**: 既存PRなしとして扱い、ステップ6に進む
  - 上記いずれにも該当しない場合（既存PRが存在しない場合）: ステップ6に進む
6. `commit-push-pr` スキルでPRを作成する
7. `/fix-pr` スキルでCI待機・レビュー対応・マージを行う
  - 引数にPR番号を渡す
8. `/retro` スキルで振り返りを行う
