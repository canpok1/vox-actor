Feature: viewer グレースフルシャットダウン

  Background:
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている
    And viewer が起動してHTTPサーバーが待ち受け中である

  Scenario: SIGINT 受信時に viewer がグレースフルシャットダウンする
    When viewer プロセスに SIGINT を送信する
    Then 5秒以内に viewer プロセスが終了する
    And 終了コードが 0 である
    # test: test/e2e/viewer_shutdown_test.go::TestViewerE2E_SIGINT_GracefulShutdown

  Scenario: SIGTERM 受信時に viewer がグレースフルシャットダウンする
    When viewer プロセスに SIGTERM を送信する
    Then 3秒以内に viewer プロセスが正常終了する
    And 終了コードが 0 である
    # test: test/e2e/viewer_shutdown_test.go::TestViewerE2E_SIGTERM_GracefulShutdown
