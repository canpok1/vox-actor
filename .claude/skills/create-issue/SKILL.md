---
name: create-issue
description: ユーザーが `/create-issue` で手動実行した場合のみ使用。ユーザーとの会話で仕様を整理し、GitHub Issueを作成する。実装は行わない。
context: fork
agent: general-purpose
allowed-tools: Bash(gh issue create *), Read, Grep, Glob, WebSearch, WebFetch, AskUserQuestion
disable-model-invocation: false
user-invocable: true
argument-hint: "[topic]"
---

## 役割

ユーザーと会話しながら仕様を整理し、GitHub Issueを作成します。
実装は行いません。Issue作成のみに専念します。

## 禁止事項

- **コードの実装・修正を絶対に行わないこと**（ファイルの編集、Write、Edit ツールの使用は禁止）
- Issue作成後に「続けて実装しましょう」等と提案しないこと
- ブランチの作成・切り替えを行わないこと
- `ready` ラベルを付与しないこと（`ready` ラベルはユーザーがIssue内容を確認後に手動で付与するもの）
- ラベルの付与はユーザーの明示的な指示がある場合のみとすること

## ワークフロー

### 1. ヒアリング

引数にトピックが指定されていれば、それを起点に会話を始める。
指定がなければ、何について Issue を作成したいか確認する。

`AskUserQuestion` を使って以下を確認：
- 何を実現したいか（目的・ゴール）
- 背景・動機（なぜ必要か）
- 制約・考慮事項（あれば）

### 2. 調査

必要に応じてコードベースを調査し、実現可能性や影響範囲を把握する。

```bash
# 関連ファイルの検索
# Glob, Grep, Read で調査
```

### 3. 仕様整理

会話の内容を Issue の形にまとめる。以下の構成を基本とする：

```markdown
## 概要
（何をするか、1〜2文で）

## 背景
（なぜ必要か）

## やりたいこと
（具体的な要件・受け入れ条件）

## 実装方針（任意）
（技術的なアプローチや注意点があれば）
```

### 4. 確認

`AskUserQuestion` でユーザーにドラフト内容を提示し、承認を得る。
修正依頼があれば内容を調整して再度確認する。

### 5. Issue 作成

承認を得たら `gh issue create` で Issue を作成する。

```bash
gh issue create --repo {owner}/{repo} \
  --title "タイトル" \
  --body "$(cat <<'EOF'
## 概要
...
EOF
)"
```

作成後、Issue の URL をユーザーに報告する。

### 6. 完了

Issue の URL を報告したら、このスキルの作業は**完了**とする。
実装には進まず、ここで終了すること。
