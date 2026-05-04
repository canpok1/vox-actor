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

  # ===== 成功・失敗シナリオ =====

  Scenario: 音声デバイス open 成功時に exit 0 が返る（音声デバイスが存在する環境）
    Given 音声デバイスが利用可能な環境である
    When "vox-actor audio-check" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が空である
    And 標準エラー出力が空である
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_DeviceAvailable_ExitZero

  Scenario: 音声デバイス open 成功時に --verbose で stderr に "audio backend: <name>" が出力される
    Given 音声デバイスが利用可能な環境である
    When "vox-actor audio-check --verbose" を実行する
    Then 終了コード 0 で完了する
    And stderr に "audio backend:" が含まれる
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_VerboseBackend_Success

  Scenario: 音声デバイス open 失敗時に exit 1 が返る（音声デバイスが存在しない環境）
    Given 音声デバイスが利用不可な環境である
    When "vox-actor audio-check" を実行する
    Then 終了コード 1 で完了する
    And 標準出力が空である
    And 標準エラー出力が空である
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_DeviceUnavailable_ExitOne

  Scenario: 音声デバイス open 失敗時に --verbose で stderr に失敗理由が出力される
    Given 音声デバイスが利用不可な環境である
    When "vox-actor audio-check --verbose" を実行する
    Then 終了コード 1 で完了する
    And stderr に "Error:" が含まれる
    # test: test/e2e/audio_check_test.go::TestAudioCheckE2E_VerboseError_Failure
