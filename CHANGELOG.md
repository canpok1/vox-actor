# Changelog

## Unreleased

### New Features

- **`--viewer-url` / `VOX_VIEWER_URL` オプション追加** ([#481](https://github.com/canpok1/vox-actor/issues/481))
  - `say` / `act` / `watch` にリモート viewer の URL を明示指定する `--viewer-url` オプションを追加しました。
  - `VOX_VIEWER_URL` 環境変数でデフォルト値を設定できます。
  - 指定時は lockfile auto-detect をスキップして明示 URL の viewer に POST `/api/play` します。
  - 接続失敗時はローカル再生にフォールバックせずエラー終了します（終了コード 1）。
  - `--viewer-url` 指定時はローカルの `AudioProbe` をスキップします（音声デバイスなし環境でもリモート再生可能）。
- **`watch` に viewer lockfile auto-detect を追加** ([#481](https://github.com/canpok1/vox-actor/issues/481))
  - `watch` コマンドが `say` / `act` と同様に `~/.vox-actor/viewer/viewer.lock` で viewer を自動検知し、起動中の viewer があれば POST `/api/play` 経由で再生するようになりました。

### Breaking Changes

- **セリフファイルフィールド名の変更** ([#474](https://github.com/canpok1/vox-actor/issues/474))
  - `.json` / `.jsonl` ファイルのフィールド名を CLI 引数と揃えました。
  - `speedScale` → `speed`
  - `pitchScale` → `pitch`
  - `intonationScale` → `intonation`
  - 旧フィールド名を含むファイルは読み込みエラーとなります（後方互換なし）。
