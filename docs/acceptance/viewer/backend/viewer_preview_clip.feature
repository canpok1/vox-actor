Feature: viewer バックエンド プレビュークリップAPI

  # ── viewer_preview_clip_test.go ────────────────────────────────

  Scenario: GET /api/preview-clip が 405 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /api/preview-clip を送信する
    Then ステータスコード 405 が返る
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_MethodNotAllowed

  Scenario: voicevoxClient 未設定時に POST /api/preview-clip が 503 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する（voicevox クライアント未設定）
    And POST /api/preview-clip に {"text":"テスト","speaker_id":3} を送信する
    Then ステータスコード 503 が返る
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_NoClient

  Scenario: 不正な JSON ボディを POST /api/preview-clip に送ると 400 が返る
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/preview-clip に不正な JSON "not-json" を送信する
    Then ステータスコード 400 が返る
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_InvalidJSON

  Scenario: text が空のリクエストで POST /api/preview-clip が 400 を返す
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/preview-clip に {"text":"","speaker_id":3} を送信する
    Then ステータスコード 400 が返る
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_EmptyText

  Scenario: 正常リクエストで POST /api/preview-clip が 200 と WAV バイト列を返す
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/preview-clip に {"text":"テスト","speaker_id":3} を送信する
    Then ステータスコード 200 が返る
    And Content-Type が "audio/wav" である
    And レスポンスボディが空でない（WAV バイト列）
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_Success

  Scenario: VOICEVOX 合成エラー時に POST /api/preview-clip が 502 を返す
    Given vox-actor バイナリがビルド済みである
    And /audio_query エンドポイントが 500 を返すフェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/preview-clip に {"text":"テスト","speaker_id":3} を送信する
    Then ステータスコード 502 が返る
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_SynthesisFails

  Scenario: POST /api/preview-clip は SSE clip イベントを配信しない
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And /events に SSE 接続する
    And POST /api/preview-clip に {"text":"テスト","speaker_id":3} を送信する
    Then SSE ストリームに "clip" イベントが配信されない（300ms 待機）
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_NoSSEBroadcast

  Scenario: POST /api/preview-clip は履歴ファイルへ追記しない
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/preview-clip に {"text":"テスト","speaker_id":3} を送信する
    Then 履歴ファイル（YYYY-MM-DD.jsonl）が作成されない
    # test: internal/infra/http_stream_player_test.go::TestHTTPStreamPlayer_APIPreviewClip_NoHistoryWrite
