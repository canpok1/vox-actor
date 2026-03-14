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

1. Issue の内容を理解する
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
    2. **open状態のPRが存在する場合**: 新しいPRを作成せず、既存PRに対してステップ7以降（CI待機・レビュー対応・マージ）を継続する
       - 複数のopen PRがある場合は、最新のものを対象とする
       - PR番号の取得例: `gh pr list --repo {owner}/{repo} --search "issue-{番号}" --state open --json number --jq '.[0].number'`
    3. **closed状態（マージされずにクローズ）のPRのみ存在する場合**: 既存PRなしとして扱い、ステップ6に進む
  - 上記いずれにも該当しない場合（既存PRが存在しない場合）: ステップ6に進む
6. `commit-push-pr` スキルでPRを作成する
7. CIの終了を待機する
  - コマンド: `gh pr checks --repo {owner}/{repo} {PR番号} --watch`
8. レビューコメントの到着をポーリングで検知する
  - CI完了後、AIレビュワー（CodeRabbit等）のレビューが到着するまでポーリングする
  - ポーリング方法:
    1. 以下のコマンドでCodeRabbitのレビュー・コメントの存在を確認する
       - コメント: `gh pr view --repo {owner}/{repo} {PR番号} --json comments --jq '[.comments[] | select(.author.login=="coderabbitai")] | length'`
       - レビュー: `gh pr view --repo {owner}/{repo} {PR番号} --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | length'`
    2. いずれかが1件以上存在すれば、レビュー到着と判定して次のステップに進む
    3. 0件の場合は30秒待機して再チェックする
    4. 最大10回（約5分）までポーリングを繰り返す
    5. タイムアウトした場合（10回チェックしてもレビューが0件の場合）は、レビュー未到着のまま次のステップに進む
       - ステップ9では `rate limit` 文言が明示的に検出された場合のみ再レビュー要求フローに入る
       - `rate limit` が検出されない場合は、そのままステップ10へ進む
9. AIレビュワーのrate limitチェックを行う
  - PRのコメントおよびレビュー本文を確認し、AIレビューの有無と `rate limit` 通知をチェックする
    - コメント: `gh pr view --repo {owner}/{repo} {PR番号} --json comments --jq '.comments[] | select(.author.login=="coderabbitai") | {body: .body, createdAt: .createdAt}'`
    - レビュー本文: `gh pr view --repo {owner}/{repo} {PR番号} --json reviews --jq '.reviews[] | select(.author.login=="coderabbitai") | {body: .body, submittedAt: .submittedAt}'`
    - AIレビューの判定基準: 投稿者が `coderabbitai` であるコメント・レビューをAIレビューとしてカウントする
    - 以前の `rate limit` コメントが残っていても、その後に正常なAIレビュー完了が確認できる場合は未検出として扱う（rate limitコメントの `createdAt` より新しいAIレビューが存在するかで判断する）
  - AIレビューのコメント・レビューが**1件も存在しない**場合:
    1. 60秒待機してから、上記と同じコマンドで再度コメント・レビューをチェックする
    2. 待機後もAIレビューが0件の場合は、rate limitが原因と見なして以下のrate limit対応フローに入る
  - 現在も有効なrate limitコメントが検出された場合:
    1. コメントの内容を読み取り、待機時間や再レビュー方法を把握する
    2. `date -u` コマンドで現在時刻を確認し、待機時間の解除タイミングと比較する（推測で「既に過ぎているはず」と判断してはならない）
    3. 待機時間がまだ経過していない場合は、残りの時間だけ待機する（情報が不明な場合は10分をデフォルトとする）
    4. `@coderabbitai full review` コメントを投稿して再レビューを要求する（`@coderabbitai review` はrate limitでスキップされたコミットの再レビューに対応していないため、必ず `full review` を使用すること）
    5. 再度CIの終了を待機する（手順7に戻る）
  - rate limitが検出されない場合は次の手順に進む
10. `/pr-comments` スキルでレビューコメントを取得し、必要に応じてコードを修正する
  - コードを修正した場合はコミット・プッシュを行いレビューコメントに返信して、手順7に戻る
  - レビューコメントへの返信時は、レビュースレッド内の全レビュワーに対してメンションすること
11. PRをマージする
  - マージ前にリポジトリの許可されたマージ方式を確認する
    - コマンド: `gh api repos/{owner}/{repo} --jq '{mergeCommit: .allow_merge_commit, squash: .allow_squash_merge, rebase: .allow_rebase_merge}'`
  - 許可されたマージ方式に基づいて、以下の優先順位でマージオプションを選択する
    1. `squash` が許可されている場合: `--squash`
    2. `merge` が許可されている場合: `--merge`
    3. `rebase` が許可されている場合: `--rebase`
    4. いずれも許可されていない場合: エラーメッセージを表示し、リポジトリ設定の確認を促して処理を中断する
  - コマンド: `gh pr merge --repo {owner}/{repo} {PR番号} {選択したマージオプション}`
  - **`--delete-branch` オプションを使用しないこと**（worktree環境ではデフォルトブランチへの切り替えに失敗しエラーになるため）
  - リモートブランチの削除はGitHub側のブランチ自動削除設定に任せる
  - マージできない場合は、原因を確認して必要に応じてコードを修正し、手順7に戻る
12. `/retro` スキルで振り返りを行う
