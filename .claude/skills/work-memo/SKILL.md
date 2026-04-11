---
name: work-memo
description: 作業メモファイルへの記録・参照方法を提供するスキル。長時間の作業を行う際に各ステップでメモファイルを作成・追記したり、過去の作業を振り返る際にメモファイルを参照する場合に使用する。
agent: general-purpose
allowed-tools: Bash(.claude/skills/work-memo/scripts/get-memo-path.sh), Bash(.claude/skills/work-memo/scripts/move-memo.sh *), Read, Write, Edit
---

## メモファイルのパスの取得

現在のブランチに対応するメモファイルのパスを取得する:

```bash
.claude/skills/work-memo/scripts/get-memo-path.sh
```

標準出力に絶対パスを出力する。メモディレクトリは自動作成される。
ファイルが存在しない場合は、過去のメモがないものとして扱う。

## メモファイルの移動（done/・issued/ へのアーカイブ）

完了・処理済みとなったメモを `${WORKSPACE_DIR}/.tmp/memo/<サブディレクトリ>/` へ
衝突回避付きで移動する:

```bash
.claude/skills/work-memo/scripts/move-memo.sh <src-abs-path> <dest-subdir>
```

- `<src-abs-path>`: 移動元のメモファイル絶対パス
- `<dest-subdir>`: `done` または `issued`（`retro` スキルは `done`、`memo-to-issue` スキルは `issued` を指定する）
- 同名ファイルが既に存在する場合はタイムスタンプ付き（`<basename>.YYYYMMDDHHMMSS.md`）に自動リネームされる
- 標準出力に移動先の絶対パスが1行出力される

## メモファイルのテンプレート

新規作成時は以下のテンプレートに従う。

```markdown
# {作業タイトル}

## 目的

{作業の目的を要約}

## 作業内容

- [ ] {タスク1}
- [ ] {タスク2}

## 作業ログ

### ステップ1: {ステップ名} (YYYY-MM-DD HH:MM)
- {内容}
```

## 書き込みルール

- 既存ファイルがある場合は Read で読み取ってから Write で追記する（上書き禁止）
- 最初に書き込む時は目的・作業内容チェックリストを設定する
- 各ステップ完了時に作業ログセクションへ追記する
- 作業内容チェックリストは完了時にチェックを付ける

