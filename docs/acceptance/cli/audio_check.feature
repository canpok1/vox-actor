Feature: vox-actor audio-check コマンド

  Scenario: 予期しない位置引数を渡すと終了コード 2 になる
    When "vox-actor audio-check unexpected" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_UnexpectedArg_ExitCode2

  Scenario: 存在しないフラグを渡すと終了コードが 0 以外になる
    When "vox-actor audio-check --bogus" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_UnknownFlag_NonZeroExit

  Scenario: "--help" フラグで使用方法と "--verbose" が表示される
    When "vox-actor audio-check --help" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "audio-check" と "--verbose" が含まれる
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_Help_ExitZeroWithUsage
