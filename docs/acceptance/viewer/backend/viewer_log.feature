Feature: viewer 起動ログ

  Scenario: VOICEVOX 接続成功時に "speakers loaded" ログが出力される
    Given フェイク VOICEVOX サーバーが起動している
    When viewer を VOX_ENGINE_URL にフェイクサーバーを指定して起動する
    Then stderr に "speakers loaded" ログが出力される
    And そのログに "speakerCount" または "styleCount" が含まれる
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_log_test.go::TestViewerE2E_Log_SpeakersLoaded

  Scenario: VOICEVOX 不在時にサイレントモード移行の WARN ログが出力される
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている
    When viewer を起動する
    Then stderr に "VOICEVOX engine unreachable" ログが出力される
    And そのログに "silent mode" が含まれる
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_log_test.go::TestViewerE2E_Log_SilentModeWarn
