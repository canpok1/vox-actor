Feature: viewer サイレントモード

  Background:
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている

  Scenario: サイレントモード移行時に WARN ログと silent=true が出力される
    When viewer を起動する
    Then stderr に "VOICEVOX engine unreachable" ログが出力される
    And stderr の "stream server listening" ログに "silent=true" が含まれる
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_silent_test.go::TestViewerE2E_SilentMode_WarnLog

  Scenario: /api/status レスポンスにサイレントモードのフィールドが含まれる
    When viewer を起動して /api/status にリクエストする
    Then HTTP ステータス 200 が返る
    And レスポンス JSON の "silent" フィールドが true である
    And レスポンス JSON の "silentReason" フィールドが空でない
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_silent_test.go::TestViewerE2E_SilentMode_StatusFields
