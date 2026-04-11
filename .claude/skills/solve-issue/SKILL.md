---
name: solve-issue
description: ユーザーが `/solve-issue` で手動実行した場合のみ使用。GitHub Issueの対応を行うスキル。実装、自己レビュー、PR作成、マージ、振り返りまで一連の流れを一気に行う。
agent: general-purpose
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: true
user-invocable: true
argument-hint: "[issue-number]"
---

GitHub Issue $ARGUMENTS を対応します。

0. Issue の状態を確認する
  - `gh issue view $ARGUMENTS --json state --jq .state` で対象IssueのOPEN/CLOSED状態を確認する
  - `CLOSED` の場合: 「Issue #$ARGUMENTS は既にクローズ済みです」と報告して処理を終了する
  - `OPEN` の場合: 次のステップに進む
1. Issue の内容を理解する
2. 実装する
  - Issueの内容からGoコードの変更が必要かどうかを判断する
    - Goコードの変更を含む場合: `/tdd` スキルで実装する
    - Goコードの変更を含まない場合（例: `.md` ファイルのみの変更）: `/tdd` をスキップし、直接実装する
3. `/review` スキルで自己レビュー（コード品質 + ドキュメント整合性チェック）を行う
4. lint/formatチェックを実行する（PR作成前の最終ガード）
  - `gofmt -l .` → 出力があれば `gofmt -w .` で修正
  - `golangci-lint run` → 指摘があれば修正
  - `shellcheck scripts/*.sh` → 指摘があれば修正
  - 修正した場合はコミットする
5. 同一Issueに対する既存PRの重複チェックを行う
  - ブランチ名検索（第一手段）: `gh pr list --repo {owner}/{repo} --head "worktree-issue-{番号}" --state all`
  - テキスト検索（フォールバック）: `gh pr list --repo {owner}/{repo} --search "#{番号}" --state all`
  - 両方の結果を合わせて、以下の優先順位で判断する:
    1. **merged状態のPRが存在する場合**: 既に対応済みのため、以下の手順でIssueをクローズし、処理をスキップして完了する（ステップ9の振り返りのみ実施する）
       - merged PRの情報を取得: `gh pr list --repo {owner}/{repo} --search "#{番号}" --state merged --json number,url --jq '.[0]'`
       - Issueをコメント付きでクローズ: `gh issue close {番号} --repo {owner}/{repo} --comment "対応済みPR #{PR番号} が既にマージされているためクローズします。\n\nPR: {PR URL}"`
    2. **open状態のPRが存在する場合**: 新しいPRを作成せず、既存PRに対してステップ7（fix-pr）を継続する
       - 複数のopen PRがある場合は、最新のものを対象とする
       - PR番号の取得例: `gh pr list --repo {owner}/{repo} --head "worktree-issue-{番号}" --state open --json number --jq '.[0].number'`
    3. **closed状態（マージされずにクローズ）のPRのみ存在する場合**: 既存PRなしとして扱い、ステップ6に進む
  - 上記いずれにも該当しない場合（既存PRが存在しない場合）: ステップ6に進む
6. `commit-push-pr` スキルでPRを作成する
7. `/fix-pr` スキルでCI待機・レビュー対応・マージを行う
  - 引数にPR番号を渡す
8. `/retro` スキルで振り返りを行う
