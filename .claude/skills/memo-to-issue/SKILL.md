---
name: memo-to-issue
description: retro完了済みの作業メモ（memo/done配下）から改善案を抽出し、GitHub Issueとして作成するスキル。処理済みメモはmemo/issuedへ移動する。
user-invocable: true
---

`retro` スキルによって完了済みとなった作業メモに記録された改善案をGitHub Issue化します。
実装は行いません。Issue作成とメモファイルの移動のみに専念します。

## 前提となるメモのライフサイクル

作業メモは以下のディレクトリを遷移する：

1. `${WORKSPACE_DIR}/.tmp/memo/` — 進行中の作業メモ
2. `${WORKSPACE_DIR}/.tmp/memo/done/` — `retro` スキルにより振り返り記録済み（本スキルの入力）
3. `${WORKSPACE_DIR}/.tmp/memo/issued/` — 本スキルによりIssue化検討済み（出力先）

本スキルは **`memo/done/` 配下のメモのみを処理対象とする**。`memo/` 直下のメモは進行中とみなし、絶対に触らない。

## 禁止事項

- **コードの実装・修正を絶対に行わないこと**（作業メモの移動とGitHub Issue作成以外のファイル編集は禁止）
- **コミット・プッシュを行わないこと**
- **ブランチの作成・切り替えを行わないこと**
- **`memo/` 直下のメモは絶対に読んだり移動したりしないこと**（retro未完了のため）
- **`ready` ラベルを付与しないこと**（`ready` ラベルはユーザーがIssue内容を確認後に手動で付与するもの）
- Issue作成後に「続けて実装しましょう」等と提案しないこと

## ワークフロー

### 1. 対象メモの列挙

`memo/done/` 直下の `.md` ファイルを列挙する：

```bash
DONE_DIR="${WORKSPACE_DIR}/.tmp/memo/done"
if [[ -d "$DONE_DIR" ]]; then
    find "$DONE_DIR" -maxdepth 1 -type f -name '*.md' | sort
fi
```

- ディレクトリが存在しない、または該当ファイルがない場合は「処理対象の完了メモはありません」と報告して終了する

### 2. 既存Issue一覧の取得（重複チェック用・1回のみ）

ステップ4でのN+1呼び出しを避けるため、ここで一度だけ既存Issueを取得してローカルに保持する：

```bash
gh issue list --state all --limit 200 --json number,title,body,labels
```

- 取得結果は以降のステップで各改善案と照合する
- リポジトリは gh のカレントリポジトリ自動判定に任せる（`--repo` 指定不要）

### 3. 各メモの読み込みと改善案の抽出

ステップ1で取得したメモファイルを Read で読み込み、振り返り結果（スムーズだった点・問題点・改善案）を抽出する。

- 改善案が記録されていないメモはIssue化をスキップする（ただしステップ5で `issued/` へ移動は行う）
- 改善案には、ルール（`.claude/rules/`）やスキル定義（`.claude/skills/`）への反映対象ファイルパスを含めること

### 4. GitHub Issueの作成

ステップ2で取得した既存Issue一覧と照合し、重複していない改善案をIssue化する。

- 類似タイトル・類似本文のIssueが既存一覧にある場合は新規作成しない
- 判断に迷う場合は作成せず、報告時にその旨を記載する
- 粒度は **1改善案あたり1Issue**
- Issue本文には対象となるルール・スキル定義ファイルのパスを明記する
- Issue本文の末尾に以下のフッターを必ず付与する：
  ```
  ---
  🤖 Generated with [Claude Code](https://claude.ai/claude-code)
  ```
- Issue作成例：
  ```bash
  gh issue create \
    --title "タイトル" \
    --body "$(cat <<'EOF'
  ## 概要
  ...

  ## 背景
  `retro` スキルによる振り返りで抽出された改善案。

  ## 対象ファイル
  - `.claude/skills/xxx/SKILL.md`
  - `.claude/rules/yyy.md`

  ---
  🤖 Generated with [Claude Code](https://claude.ai/claude-code)
  EOF
  )"
  ```
- 作成後、Issueのレイアウトが崩れていないか・フッターが付与されているかを確認する

### 5. 処理済みメモを issued/ へ移動

改善案の有無・Issue作成の有無にかかわらず、ステップ1で列挙した全メモを `memo/issued/` へ移動する。`work-memo` スキルの移動ヘルパーを使用する：

```bash
.claude/skills/work-memo/scripts/move-memo.sh "<対象メモの絶対パス>" issued
```

- スクリプトが `${WORKSPACE_DIR}/.tmp/memo/issued/` を自動作成し、同名衝突時はタイムスタンプ付き（`<basename>.YYYYMMDDHHMMSS.md`）でリネームする
- 標準出力に移動先の絶対パスが出力される

### 6. 結果の報告

ユーザーに以下を報告する：

- 作成したIssueのURL一覧（件数・タイトル付き）
- スキップした改善案と理由（既存Issueと重複など）
- `issued/` へ移動したメモファイル一覧
- 改善案がなかったためIssue化せず移動のみ行ったメモ

## 完了条件

- Issue作成とメモファイル移動の完了、および結果報告をもって**完了**とする
- 実装には進まず、ここで終了すること
