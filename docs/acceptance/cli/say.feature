Feature: vox-actor say コマンド

  # ===== ドライラン基本動作 =====

  Scenario: ドライランで発話テキストを渡すと合成完了・再生完了ログが出力される
    When "vox-actor say --dry-run こんにちは" を実行する
    Then 終了コード 0 で完了する
    And stderr に "[dry run] synthesis completed" が含まれる
    And stderr に "[dry run] playback completed" が含まれる
    And stderr に "text=こんにちは" が含まれる
    # test: test/e2e/say_test.go::TestSayE2E_DryRun_Basic

  Scenario: ドライランでフラグを指定するとパラメータがプレイバックログに反映される
    When "vox-actor say --dry-run --speaker 8 --speed 1.2 --pitch 0.1 --intonation 1.5 テスト" を実行する
    Then 終了コード 0 で完了する
    And "[dry run] playback completed" の行に "text=テスト", "speaker=8", "speed=1.2", "pitch=0.1", "intonation=1.5" が含まれる
    # test: test/e2e/say_test.go::TestSayE2E_DryRun_Flags

  Scenario: VOX_SPEAKER 環境変数でスピーカーを指定するとプレイバックログに反映される
    Given 環境変数 VOX_SPEAKER=11 が設定されている
    When "vox-actor say --dry-run 環境変数" を実行する
    Then 終了コード 0 で完了する
    And "playback completed" 行に "speaker=11" が含まれる
    # test: test/e2e/say_test.go::TestSayE2E_DryRun_EnvVarVOXSpeaker

  Scenario: ドライラン中は到達不能な VOX_ENGINE_URL を指定しても正常終了する
    Given 環境変数 VOX_ENGINE_URL=http://example.invalid:50021 が設定されている
    When "vox-actor say --dry-run エンジンURL" を実行する
    Then 終了コード 0 で完了する
    And stderr に "[dry run] playback completed" が含まれる
    # test: test/e2e/say_test.go::TestSayE2E_DryRun_EnvVarVOXEngineURL_NotFailing

  # ===== 引数バリデーション =====

  Scenario Outline: 引数の数が不正な場合は終了コード 2 が返る
    When <引数> で "vox-actor say" を実行する
    Then 終了コード 2 で完了する

    Examples:
      | 引数             |
      | 引数なし         |
      | 引数が2つ以上    |
    # test: test/e2e/say_test.go::TestSayE2E_ArgValidation_ExitCode2

  # ===== 詳細ログ =====

  Scenario: --verbose フラグを付けると通常より多くのログ行が出力される
    Given "--dry-run 詳細ログ" でのベースライン行数を計測する
    When "vox-actor say --dry-run --verbose 詳細ログ" を実行する
    Then 終了コード 0 で完了する
    And ログ行数がベースラインより多い
    # test: test/e2e/say_test.go::TestSayE2E_Verbose_IncreasesLogs

  # ===== VOICEVOX 実通信エラー =====

  Scenario: VOICEVOX が 5xx を返すと非ゼロ終了コードで "500" または "audio" が stderr に出力される
    Given VOICEVOX のスタブサーバーが常に 500 を返す
    When "vox-actor say テスト" を VOX_ENGINE_URL に向けて実行する
    Then 終了コードが非ゼロである
    And stderr に "500" または "audio" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_RealComm_Voicevox5xx_NonZeroExit

  Scenario: VOICEVOX への接続が失敗すると非ゼロ終了コードと stderr が出力される
    Given 環境変数 VOX_ENGINE_URL=http://127.0.0.1:1 が設定されている
    When "vox-actor say テスト" を実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/say_comm_test.go::TestSayE2E_RealComm_VoicevoxConnFailed_NonZeroExit

  Scenario: VOICEVOX が /audio_query に不正 JSON を返すと非ゼロ終了コードと stderr が出力される
    Given /audio_query が "not-json" を返すスタブサーバーを用意する
    When "vox-actor say テスト" を VOX_ENGINE_URL に向けて実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/say_comm_test.go::TestSayE2E_RealComm_VoicevoxBadBody_NonZeroExit

  # ===== viewer 連携 =====

  Scenario: viewer が起動中の場合、/api/play に1回 POST して終了コード 0 が返る
    Given フェイク viewer サーバーが起動しロックファイルが書き込まれている
    When "vox-actor say こんにちは" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And /api/play への POST が1回だけ呼ばれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Viewer_POSTsAndExitsZero

  Scenario: viewer がサイレントモードの場合、終了コード 0 かつ WARN ログが出力される
    Given フェイク viewer がサイレントモード（理由: "test-reason"）で起動している
    When "vox-actor say テスト" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And stderr の "viewer is in silent mode" 行が WARN レベルプレフィックス "!" で始まる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Viewer_SilentMode_Warns

  Scenario: viewer と VOICEVOX の両方に到達できない場合、非ゼロ終了コードと "viewer not running" ログが出力される
    Given ロックファイルが到達不能アドレス "127.0.0.1:1" を指している
    And VOX_ENGINE_URL も "http://127.0.0.1:1" に設定する
    When "vox-actor say --verbose テスト" を実行する
    Then 終了コードが非ゼロである
    And stderr に "viewer not running" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Viewer_Unreachable_FallbackLogged

  Scenario: --dry-run 時は viewer への POST を行わない
    Given フェイク viewer サーバーが起動している
    When "vox-actor say --dry-run テスト" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And /api/play への POST が0回である
    # test: test/e2e/say_comm_test.go::TestSayE2E_Viewer_DryRun_NoPOST

  Scenario: viewer 起動中は "viewer detected" ログが出力される
    Given フェイク viewer サーバーが起動している
    When "vox-actor say テスト" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And stderr に "viewer detected" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Viewer_Detected_LogMessage

  # ===== 環境変数とフラグの優先順位 =====

  Scenario: --speaker フラグが VOX_SPEAKER 環境変数より優先される
    Given 環境変数 VOX_SPEAKER=10 が設定されている
    When "vox-actor say --dry-run --speaker 5 テスト" を実行する
    Then 終了コード 0 で完了する
    And "playback completed" 行に "speaker=5" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Flag_SpeakerOverridesEnv

  Scenario: --engine-url フラグが VOX_ENGINE_URL 環境変数より優先される
    Given 環境変数 VOX_ENGINE_URL に到達不能アドレスを設定する
    And --engine-url に 5xx を返すスタブサーバーの URL を指定する
    When "vox-actor say --engine-url <stub_url> テスト" を実行する
    Then 終了コードが非ゼロである
    And stderr に "500" または "audio" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Flag_EngineURLOverridesEnv

  Scenario: VOX_SPEAKER に不正な値を設定するとデフォルト値（speaker=3）にフォールバックする
    Given 環境変数 VOX_SPEAKER=not-a-number が設定されている
    When "vox-actor say --dry-run テスト" を実行する
    Then 終了コード 0 で完了する
    And "playback completed" 行に "speaker=3" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Env_VOXSpeakerInvalid_DefaultFallback

  # ===== ヘルプ =====

  Scenario: --help フラグで終了コード 0 かつ全フラグが表示される
    When "vox-actor say --help" を実行する
    Then 終了コード 0 で完了する
    And stdout に "--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Help_ExitZeroWithAllFlags

  # ===== 空文字・境界値 =====

  Scenario: 空文字列を渡してドライランすると終了コード 0 かつ "playback completed" が出力される
    When "vox-actor say --dry-run ''" を実行する
    Then 終了コード 0 で完了する
    And stderr に "playback completed" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_EmptyString_DryRun_ExitsZero

  Scenario Outline: 境界値パラメータを指定してドライランすると終了コード 0 が返る
    When "vox-actor say --dry-run <フラグ> テスト" を実行する
    Then 終了コード 0 で完了する

    Examples:
      | フラグ                      |
      | --speaker -1               |
      | --speed 999.0              |
      | --pitch 1.0                |
      | --intonation 999.0         |
    # test: test/e2e/say_comm_test.go::TestSayE2E_BoundaryValues_DryRun_ExitsZero

  # ===== シグナル処理 =====

  Scenario: VOICEVOX 処理中に SIGINT を送ると 5 秒以内にプロセスが終了する
    Given ハングするスタブ VOICEVOX サーバーを起動する
    And "vox-actor say --engine-url <stub_url> テスト" をバックグラウンドで起動する
    When 150ms 後に SIGINT を送る
    Then 5 秒以内にプロセスが終了する
    # test: test/e2e/say_comm_test.go::TestSayE2E_SIGINT_DuringVoicevox_Exits

  Scenario: --verbose フラグで "say starting" デバッグ行にテキストが含まれる
    When "vox-actor say --dry-run --verbose テスト内容" を実行する
    Then 終了コード 0 で完了する
    And stderr の "say starting" 行に "text=テスト内容" が含まれる
    # test: test/e2e/say_comm_test.go::TestSayE2E_Verbose_ShowsSpecificDebugLogs
