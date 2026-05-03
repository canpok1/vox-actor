Feature: vox-actor playback コマンド

  # ===== 正常系 =====

  Scenario: local_playback を渡すと即時に終了コード 0 が返る
    When "vox-actor playback wait local_playback" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_LocalPlayback_ExitsZero

  Scenario: viewer から完了通知を受信して終了コード 0 が返る
    Given フェイク viewer サーバーが起動しロックファイルが書き込まれている
    And フェイク viewer が playback_id の status=completed を返す
    When "vox-actor playback wait <id>" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_Completed_ExitsZero

  Scenario: --viewer-url 明示時は lockfile auto-detect をスキップして接続する
    Given フェイク viewer サーバーが status=completed を返す
    When "vox-actor playback wait <id> --viewer-url <url>" をロックファイルなしで実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_ViewerURL_SkipsLockfile

  # ===== エラー系 =====

  Scenario: --server-down-timeout 経過後にサーバ未応答だった場合は非ゼロ終了コードが返る
    When "vox-actor playback wait <id> --viewer-url <到達不能URL> --server-down-timeout 500ms" を実行する
    Then 終了コードが非ゼロである
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_ServerDownTimeout_NonZero

  Scenario: 引数 <id> 未指定時は終了コード 2 が返る
    When "vox-actor playback wait" を引数なしで実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_NoArgs_ExitCode2

  Scenario: viewer 未起動かつ --viewer-url 未指定時はエラーが返る
    Given ロックファイルが存在しない HOME ディレクトリを用意する
    When "vox-actor playback wait <id>" を実行する
    Then 終了コードが非ゼロである
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_ViewerNotFound_Error

  # ===== ヘルプ =====

  Scenario: --help フラグで終了コード 0 かつ主要フラグが表示される
    When "vox-actor playback wait --help" を実行する
    Then 終了コード 0 で完了する
    And stdout に "--viewer-url", "--server-down-timeout" が含まれる
    # test: test/e2e/playback_test.go::TestPlaybackWaitE2E_Help_ExitZeroWithAllFlags
