Feature: viewer 起動フラグ・環境変数

  Scenario: --help が exit 0 で終了し主要フラグが表示される
    When "viewer --help" を実行する
    Then 終了コードが 0 である
    And 標準出力に "--port" が含まれる
    And 標準出力に "--host" が含まれる
    And 標準出力に "--engine-url" が含まれる
    And 標準出力に "--speaker" が含まれる
    And 標準出力に "--watch" は含まれない
    And 標準出力に "--watch-queue" は含まれない
    And 標準出力に "--delete" は含まれない
    # test: test/e2e/viewer_flags_test.go::TestViewerE2E_Help_ExitZero

  Scenario: VOX_ENGINE_URL 環境変数で指定した VOICEVOX サーバーに接続する
    Given フェイク VOICEVOX サーバーが起動している
    When --engine-url フラグを使わずに VOX_ENGINE_URL 環境変数でフェイクサーバーを指定して viewer を起動する
    Then stderr に "speakers loaded" ログが出力される
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_flags_test.go::TestViewerE2E_Env_VOXEngineURL_Applied

  Scenario: --host 0.0.0.0 を指定したとき 127.0.0.1 でアクセスできる
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている
    When "--host 0.0.0.0" を指定して viewer を起動する
    And "stream server listening" ログを確認する
    And "http://127.0.0.1:<port>/api/status" に GET リクエストを送る
    Then HTTP ステータス 200 が返る
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_flags_test.go::TestViewerE2E_Host_0000_Binds

  # ===== --engine-url / --speed / --pitch / --intonation フラグ正常系 =====

  Scenario: --engine-url フラグで指定した VOICEVOX サーバーに接続する
    Given フェイク VOICEVOX サーバーが起動している
    When --engine-url フラグを使ってフェイクサーバーを指定して viewer を起動する
    Then stderr に "speakers loaded" ログが出力される
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_flags_test.go::TestViewerE2E_Flag_EngineURL_Applied

  Scenario: --speed / --pitch / --intonation フラグを指定しても viewer が正常起動する
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている
    When "--speed 1.2 --pitch 0.05 --intonation 1.5" を指定して viewer を起動する
    And "stream server listening" ログを確認する
    Then HTTP ステータス 200 が "api/status" から返る
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_flags_test.go::TestViewerE2E_Flags_SpeedPitchIntonation_Accepted
