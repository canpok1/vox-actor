# Changelog

## Unreleased

### Breaking Changes

- **セリフファイルフィールド名の変更** ([#474](https://github.com/canpok1/vox-actor/issues/474))
  - `.json` / `.jsonl` ファイルのフィールド名を CLI 引数と揃えました。
  - `speedScale` → `speed`
  - `pitchScale` → `pitch`
  - `intonationScale` → `intonation`
  - 旧フィールド名を含むファイルは読み込みエラーとなります（後方互換なし）。
