Feature: viewer バックエンド SSE イベントストリーム

  # ── viewer_events_test.go ────────────────────────────────────────

  Scenario: /events への接続が正しいレスポンスヘッダを返す
    Given vox-actor バイナリがビルド済みである
    And VOX_ENGINE_URL が到達不能なアドレス "http://127.0.0.1:1" に設定されている（サイレントモード）
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /events に SSE 接続する
    Then ステータスコード 200 が返る
    And Content-Type が "text/event-stream" である
    And Cache-Control が "no-cache" である
    And Connection が "keep-alive" である
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_events_test.go::TestViewerE2E_Events_ConnectionHeaders

  Scenario: POST /api/play 後に SSE で clip イベントが配信される
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And /events に SSE 接続する
    And 100ms 待機して subscriber 登録を確保する
    And POST /api/play に {"clips": [{"text": "テスト発話", "speaker_id": 3}]} を送信する
    Then ステータスコード 200 が返る
    And SSE ストリームに "clip" イベントが配信される
    And clip イベントの data.text が "テスト発話" である
    And clip イベントの data.speakerName が空でない
    And clip イベントの data.url が空でない
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_events_test.go::TestViewerE2E_Events_ClipEventAfterAPIPlay

  Scenario: 複数クライアントが同時に SSE 接続している場合に全員が clip イベントを受信する
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And 3 クライアントが /events に SSE 接続する
    And 200ms 待機して全クライアントの subscriber 登録を確保する
    And POST /api/play に {"clips": [{"text": "マルチキャストテスト", "speaker_id": 3}]} を送信する
    Then POST /api/play がステータスコード 200 を返す
    And 3 クライアント全員が "clip" イベントを受信する
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_events_test.go::TestViewerE2E_Events_MultipleClients

  Scenario: サーバーシャットダウン時に SSE 接続が切断される
    Given vox-actor バイナリがビルド済みである
    And VOX_ENGINE_URL が到達不能なアドレス "http://127.0.0.1:1" に設定されている（サイレントモード）
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And /events に SSE 接続する
    And SIGTERM を送信する（Shutdown() → subscribers.closeAll() が呼ばれる）
    Then SSE 接続が EOF またはエラーで切断される（5 秒以内）
    # test: test/e2e/viewer_events_test.go::TestViewerE2E_Events_ServerShutdownClosesSSE

  Scenario: サイレントモードで POST /api/play しても SSE に clip イベントが配信されない
    Given vox-actor バイナリがビルド済みである
    And VOX_ENGINE_URL が到達不能なアドレス "http://127.0.0.1:1" に設定されている（サイレントモード）
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する（サイレントモードになる）
    And /events に SSE 接続する
    Then /events への接続が 200 を返す
    And 100ms 待機して subscriber 登録を確保する
    When POST /api/play に {"clips": [{"text": "サイレントテスト", "speaker_id": 3}]} を送信する
    Then POST /api/play がステータスコード 200 を返す
    And レスポンス JSON の silent フィールドが true である
    And 2 秒間 SSE ストリームに "clip" イベントが配信されない
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_events_test.go::TestViewerE2E_Events_SilentMode_ConnectionAndNoClip
