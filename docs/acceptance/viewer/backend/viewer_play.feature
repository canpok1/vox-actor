Feature: viewer バックエンド 再生API・クリップ配信・テストクリップ

  # ── viewer_play_test.go ──────────────────────────────────────────

  Scenario: POST /api/play が正常に 200 と JSON を返す
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/play に {"clips": [{"text": "テスト", "speaker_id": 3}]} を送信する
    Then ステータスコード 200 が返る
    And Content-Type が "application/json; charset=utf-8" である
    And レスポンス JSON の silent フィールドが false である
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_Success

  Scenario: VOICEVOX に接続できない状態で POST /api/play がサイレントモードレスポンスを返す
    Given vox-actor バイナリがビルド済みである
    And VOX_ENGINE_URL が到達不能なアドレス "http://127.0.0.1:1" に設定されている
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/play に {"clips": [{"text": "サイレントテスト", "speaker_id": 3}]} を送信する
    Then ステータスコード 200 が返る
    And Content-Type が "application/json; charset=utf-8" である
    And レスポンス JSON の silent フィールドが true である
    And レスポンス JSON の silent_reason フィールドが空でない
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_SilentModeResponse

  Scenario: GET /api/play が 405 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /api/play を送信する
    Then ステータスコード 405 が返る
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_MethodNotAllowed

  Scenario: 不正な JSON ボディを POST /api/play に送ると 400 が返る
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/play に不正な JSON "not-json" を送信する
    Then ステータスコード 400 が返る
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_InvalidJSON

  Scenario: /audio_query が 500 を返す場合に POST /api/play が即座に 200 を返す（非同期 worker）
    Given vox-actor バイナリがビルド済みである
    And /audio_query エンドポイントが 500 を返すフェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/play に {"clips": [{"text": "テスト", "speaker_id": 3}]} を送信する
    Then ステータスコード 200 が即座に返る（非同期 worker がエラーを処理する）
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_CreateQueryFails

  Scenario: /synthesis が 500 を返す場合に POST /api/play が即座に 200 を返す（非同期 worker）
    Given vox-actor バイナリがビルド済みである
    And /synthesis エンドポイントが 500 を返すフェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And POST /api/play に {"clips": [{"text": "テスト", "speaker_id": 3}]} を送信する
    Then ステータスコード 200 が即座に返る（非同期 worker がエラーを処理する）
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_play_test.go::TestViewerE2E_APIPlay_SynthesisFails

  # ── viewer_clip_test.go ──────────────────────────────────────────

  Scenario: SSE で受信したクリップ URL が WAV を返す
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And /events に SSE 接続する
    And POST /api/play を送信する
    Then SSE ストリームに "clip" イベントが配信される
    And clip イベントの data.url が空でない
    When clip URL に GET する
    Then ステータスコード 200 が返る
    And レスポンスボディが空でない（WAV バイト列）
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_Success

  Scenario: クリップ URL のレスポンスの Content-Type が audio/wav である
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And SSE 経由でクリップ URL を取得する
    And clip URL に GET する
    Then Content-Type が "audio/wav" である
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_ContentType

  Scenario: クリップ URL のレスポンスの Cache-Control が immutable キャッシュ指定である
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And SSE 経由でクリップ URL を取得する
    And clip URL に GET する
    Then Cache-Control ヘッダが "public, max-age=31536000, immutable" である
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_CacheControl

  Scenario: タイムスタンプでない不正な ID を持つクリップ URL が 404 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /clips/not-a-timestamp.wav を送信する
    Then ステータスコード 404 が返る
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_InvalidID_404

  Scenario: 存在しないタイムスタンプのクリップ URL が 404 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /clips/1.wav を送信する（非常に過去のタイムスタンプ）
    Then ステータスコード 404 が返る
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_NonExistent_404

  Scenario: 再起動後に生成されるクリップ URL が前回と異なる
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    When 1 回目の viewer を起動して SSE 経由でクリップ URL1 を取得し停止する
    And 2 回目の viewer を別インスタンスで起動して SSE 経由でクリップ URL2 を取得する
    Then URL1 と URL2 が異なる（再起動後の衝突回避）
    # test: test/e2e/viewer_clip_test.go::TestViewerE2E_Clip_RestartCollisionAvoidance

  # ── viewer_test_clip_test.go ─────────────────────────────────────

  Scenario: GET /test-clip?speaker=<id> が WAV を返す
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /test-clip?speaker=3 を送信する
    Then ステータスコード 200 が返る
    And レスポンスボディが空でない（WAV バイト列）
    # test: test/e2e/viewer_test_clip_test.go::TestViewerE2E_TestClip_Success

  Scenario: GET /test-clip の Content-Type が audio/wav である
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /test-clip?speaker=3 を送信する
    Then Content-Type が "audio/wav" である
    # test: test/e2e/viewer_test_clip_test.go::TestViewerE2E_TestClip_ContentType

  Scenario: GET /test-clip の Content-Length がボディのバイト数と一致する
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /test-clip?speaker=3 を送信する
    Then Content-Length ヘッダが空でない
    And Content-Length の値がレスポンスボディのバイト数と一致する
    # test: test/e2e/viewer_test_clip_test.go::TestViewerE2E_TestClip_ContentLength

  Scenario: GET /test-clip を 2 回呼ぶと同一のボディが返る（冪等性）
    Given vox-actor バイナリがビルド済みである
    And フェイク VOICEVOX サーバーが起動している
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /test-clip?speaker=3 を 1 回目として送信する
    And GET /test-clip?speaker=3 を 2 回目として送信する
    Then 1 回目と 2 回目のボディが同一である
    # test: test/e2e/viewer_test_clip_test.go::TestViewerE2E_TestClip_Idempotent

  Scenario: VOICEVOX 接続不可（サイレントモード）の場合に GET /test-clip が 400 を返す
    Given vox-actor バイナリがビルド済みである
    And VOX_ENGINE_URL が到達不能なアドレス "http://127.0.0.1:1" に設定されている
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する（サイレントモードになる）
    And GET /test-clip?speaker=3 を送信する
    Then ステータスコード 400 が返る（speakerLookup が空のため speaker not found）
    # test: test/e2e/viewer_test_clip_test.go::TestViewerE2E_TestClip_SilentMode
