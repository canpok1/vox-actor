Feature: viewer ロックファイル管理

  Background:
    Given VOICEVOX エンジンが到達不能なURL（http://127.0.0.1:1）として設定されている

  Scenario: viewer 起動時にロックファイルが作成される
    When viewer を起動する
    Then "${HOME}/.vox-actor/viewer/viewer.lock" にロックファイルが存在する
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_lockfile_test.go::TestViewerE2E_Lockfile_CreatedOnStart

  Scenario: ロックファイルは HOME 配下に作成され VOX_ACTOR_WORKSPACE 配下には作成されない
    When viewer を起動する
    Then "${HOME}/.vox-actor/viewer/viewer.lock" にロックファイルが存在する
    And VOX_ACTOR_WORKSPACE ディレクトリ直下に "viewer.lock" は存在しない
    And viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_lockfile_test.go::TestViewerE2E_Lockfile_UnderHome_NotWorkspace

  Scenario: 同じ HOME で viewer を二重起動すると2番目が非ゼロ終了コードで失敗する
    Given 1番目の viewer が起動して稼働中である
    When 同じ HOME で2番目の viewer を別ポートで起動する
    Then 2番目の viewer が非ゼロ終了コードで終了する
    And 1番目の viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_lockfile_test.go::TestViewerE2E_Lockfile_DoubleStart_ErrorExit

  Scenario: 1番目の viewer 終了後に同じ HOME で再起動が成功する
    Given 1番目の viewer を起動し "stream server listening" ログを確認する
    When 1番目の viewer を正常終了させる
    And 同じ HOME で2番目の viewer を別ポートで起動する
    Then 2番目の viewer の "stream server listening" ログが出力される
    And 2番目の viewer は正常終了（exit code 0）する
    # test: test/e2e/viewer_lockfile_test.go::TestViewerE2E_Lockfile_Restart_AfterStop
