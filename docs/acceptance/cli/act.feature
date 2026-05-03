Feature: vox-actor act コマンド

  # ===== ドライラン基本動作 =====

  Scenario: .txt ファイルをドライランで実行するとファイル全体が1スクリプトとして処理される
    Given テンポラリディレクトリに "script.txt" を作成し、複数行テキストを書く
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And stderr に "[dry run] playback completed" が1回だけ出力される
    # test: test/e2e/act_test.go::TestActE2E_TxtFile_DryRun

  Scenario: .json ファイルをドライランで実行するとパラメータがログに反映される
    Given テンポラリディレクトリに JSON ファイル '{"text":"こんにちは","speaker":11,"speed":1.2,"pitch":0.1,"intonation":1.5}' を書く
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And "[dry run] playback completed" の行に "text=こんにちは", "speaker=11", "speed=1.2", "pitch=0.1", "intonation=1.5" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_JsonFile_DryRun_OverrideParams

  Scenario: .jsonl ファイルをドライランで実行すると各行が独立したスクリプトとして処理される
    Given テンポラリディレクトリに 3 行の .jsonl ファイルを書く
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And stderr に "playback completed" が3回出力される
    And テキストの出力順が "いちぎょうめ → にぎょうめ → さんぎょうめ" の順である
    # test: test/e2e/act_test.go::TestActE2E_JsonlFile_DryRun_MultipleScripts

  Scenario: ディレクトリをドライランで実行するとファイルが辞書順に処理される
    Given テンポラリディレクトリに "a.txt", "b.json", "c.jsonl" を作成する
    When "vox-actor act --dry-run <dir>" を実行する
    Then 終了コード 0 で完了する
    And stderr に "playback completed" が4回出力される（a.txt=1, b.json=1, c.jsonl=2）
    And テキストの出力順が "エーのテキスト → ビーのテキスト → シー1 → シー2" の順である
    # test: test/e2e/act_test.go::TestActE2E_Directory_DryRun_LexOrder

  # ===== エラーハンドリング =====

  Scenario: 存在しないパスを指定すると非ゼロ終了コードと stderr が出力される
    Given 存在しないファイルパスを用意する
    When "vox-actor act --dry-run <missing_path>" を実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_test.go::TestActE2E_NonexistentPath_NonZeroWithStderr

  Scenario: 壊れた JSON ファイルを指定すると非ゼロ終了コードと stderr が出力される
    Given テンポラリディレクトリに不正な JSON ファイル "{" を書く
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_test.go::TestActE2E_BrokenJSON_NonZero

  Scenario: 引数なしで実行すると終了コード 2 が返る
    When "vox-actor act" を引数なしで実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/act_test.go::TestActE2E_NoArgs_ExitCode2

  Scenario: text フィールドがない JSON ファイルを指定すると非ゼロ終了コードと stderr が出力される
    Given テンポラリディレクトリに "{}" を書いた .json ファイルを用意する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_test.go::TestActE2E_EmptyJson_NoTextField_NonZero

  # ===== フラグとパラメータの反映 =====

  Scenario: --speaker および --speed フラグの値がログに反映される
    Given テンポラリディレクトリに .txt ファイルを作成する
    When "vox-actor act --dry-run --speaker 8 --speed 1.2 <path>" を実行する
    Then 終了コード 0 で完了する
    And "[dry run] playback completed" の行に "text=フラグテスト", "speaker=8", "speed=1.2" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_TxtFile_FlagsReflectedInLog

  Scenario: JSON ファイルでパラメータを省略した場合は CLI フラグのデフォルト値が使用される
    Given テンポラリディレクトリに '{"text":"きほん"}' を書いた .json ファイルを用意する
    When "vox-actor act --dry-run --speaker 5 --speed 1.3 <path>" を実行する
    Then 終了コード 0 で完了する
    And "[dry run] playback completed" の行に "text=きほん", "speaker=5", "speed=1.3" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_JsonFile_OmittedParams_UsesCLIFlagDefaults

  Scenario: .jsonl ファイルの各行パラメータがプレイバックログに反映される
    Given テンポラリディレクトリに2行の .jsonl ファイル（各行にパラメータを指定）を作成する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And 1行目の "playback completed" に "speaker=11", "speed=1.2", "pitch=0.05", "intonation=1.3" が含まれる
    And 2行目の "playback completed" に "speaker=7" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_JsonlFile_PerLineParams_Reflected

  # ===== 廃止フラグ =====

  Scenario: 廃止済み --watch フラグを指定すると終了コード 2 が返る
    Given テンポラリディレクトリを用意する
    When "vox-actor act --watch <dir>" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/act_test.go::TestActE2E_WatchFlag_ExitCode2

  Scenario: 廃止済み --watch-delete フラグを指定すると終了コード 2 が返る
    Given テンポラリディレクトリを用意する
    When "vox-actor act --watch-delete <dir>" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/act_test.go::TestActE2E_WatchDeleteFlag_ExitCode2

  # ===== ヘルプ =====

  Scenario: --help フラグで終了コード 0 かつ全フラグが表示される
    When "vox-actor act --help" を実行する
    Then 終了コード 0 で完了する
    And stdout に "--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_Help_ExitZeroWithAllFlags

  # ===== 空ファイル・特殊ケース =====

  Scenario: 空の .txt ファイルを指定すると終了コード 0 が返る
    Given テンポラリディレクトリに空の "empty.txt" を作成する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/act_test.go::TestActE2E_EmptyFile_Txt_ExitsZero

  Scenario: 空の .jsonl ファイルを指定すると終了コード 0 が返る
    Given テンポラリディレクトリに空の "empty.jsonl" を作成する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/act_test.go::TestActE2E_EmptyFile_Jsonl_ExitsZero

  Scenario: サポート外の拡張子（.md）はプレーンテキストとして処理され終了コード 0 が返る
    Given テンポラリディレクトリに "note.md" を作成する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/act_test.go::TestActE2E_UnsupportedExtension_ExitsZero

  Scenario: .jsonl ファイルの空行はスキップされる
    Given テンポラリディレクトリに空行を含む .jsonl ファイル（有効行2行）を作成する
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And stderr に "playback completed" が2回出力される
    # test: test/e2e/act_test.go::TestActE2E_JsonlFile_EmptyLines_Skipped

  # ===== 詳細ログ =====

  Scenario: --verbose フラグでデバッグログが出力される
    Given テンポラリディレクトリに .txt ファイルを作成する
    When "vox-actor act --dry-run --verbose <path>" を実行する
    Then 終了コード 0 で完了する
    And stderr の "act starting" 行に "path=" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_Verbose_ShowsDebugLogs

  Scenario: ディレクトリを指定すると "scripts loaded" ログが出力される
    Given テンポラリディレクトリに2つの .txt ファイルを作成する
    When "vox-actor act --dry-run <dir>" を実行する
    Then 終了コード 0 で完了する
    And stderr に "scripts loaded" が含まれる
    # test: test/e2e/act_test.go::TestActE2E_Dir_ScriptsLoadedLog

  # ===== VOICEVOX 実通信エラー =====

  Scenario: VOICEVOX が 5xx を返すと非ゼロ終了コードと stderr が出力される
    Given VOICEVOX のスタブサーバーが常に 500 を返す
    When "vox-actor act <path>" を VOX_ENGINE_URL に向けて実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_comm_test.go::TestActE2E_RealComm_Voicevox5xx_NonZeroExit

  Scenario: VOICEVOX への接続が失敗すると非ゼロ終了コードと stderr が出力される
    Given VOX_ENGINE_URL に到達不能なアドレス "http://127.0.0.1:1" を設定する
    When "vox-actor act <path>" を実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_comm_test.go::TestActE2E_RealComm_VoicevoxConnFailed_NonZeroExit

  Scenario: VOICEVOX がレスポンスボディに不正 JSON を返すと非ゼロ終了コードと stderr が出力される
    Given /audio_query が "not-json" を返すスタブサーバーを用意する
    When "vox-actor act <path>" を VOX_ENGINE_URL に向けて実行する
    Then 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/act_comm_test.go::TestActE2E_RealComm_VoicevoxBadBody_NonZeroExit

  # ===== viewer 連携 =====

  Scenario: viewer が起動中の場合、全クリップを1回のバッチ POST で送信し終了コード 0 が返る
    Given フェイク viewer サーバーが起動しロックファイルが書き込まれている
    And テンポラリディレクトリに3行の .jsonl スクリプトを作成する
    When "vox-actor act <path>" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And /api/play への POST が1回だけ呼ばれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_POSTsAndExitsZero

  Scenario: viewer の /api/play が失敗すると非ゼロ終了コードが返る
    Given /api/play が常に 500 を返すスタブ viewer が起動している
    And テンポラリディレクトリに3行の .jsonl スクリプトを作成する
    When "vox-actor act <path>" を HOME に向けて実行する
    Then 終了コードが非ゼロである
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_POSTFailure_EarlyExit

  Scenario: viewer がサイレントモードの場合、終了コード 0 かつ WARN ログが出力される
    Given フェイク viewer がサイレントモード（理由: "test-reason"）で起動している
    When "vox-actor act <path>" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And stderr の "viewer is in silent mode" 行が WARN レベルプレフィックス "!" で始まる
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_SilentMode_Warns

  Scenario: viewer に到達できず VOICEVOX も到達不能な場合、非ゼロ終了コードと "viewer not running" ログが出力される
    Given ロックファイルが到達不能アドレス "127.0.0.1:1" を指している
    And VOX_ENGINE_URL も "http://127.0.0.1:1" に設定する
    When "vox-actor act --verbose <path>" を実行する
    Then 終了コードが非ゼロである
    And stderr に "viewer not running" が含まれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_Unreachable_FallbackFails

  Scenario: --dry-run 時は viewer への POST を行わない
    Given フェイク viewer サーバーが起動している
    When "vox-actor act --dry-run <path>" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And /api/play への POST が0回である
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_DryRun_NoPOST

  Scenario: viewer 起動中は "viewer detected" ログが出力される
    Given フェイク viewer サーバーが起動している
    When "vox-actor act <path>" を HOME に向けて実行する
    Then 終了コード 0 で完了する
    And stderr に "viewer detected" が含まれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Viewer_Detected_LogMessage

  # ===== 環境変数とフラグの優先順位 =====

  Scenario: --speaker フラグが VOX_SPEAKER 環境変数より優先される
    Given 環境変数 VOX_SPEAKER=10 が設定されている
    When "vox-actor act --dry-run --speaker 5 <path>" を実行する
    Then 終了コード 0 で完了する
    And "playback completed" 行に "speaker=5" が含まれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Flag_SpeakerOverridesEnv

  Scenario: --engine-url フラグが VOX_ENGINE_URL 環境変数より優先される
    Given 環境変数 VOX_ENGINE_URL に到達不能アドレスを設定する
    And --engine-url に 5xx を返すスタブサーバーの URL を指定する
    When "vox-actor act --engine-url <stub_url> <path>" を実行する
    Then 終了コードが非ゼロである
    And stderr に "500" または "audio" が含まれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Flag_EngineURLOverridesEnv

  Scenario: VOX_SPEAKER に不正な値を設定するとデフォルト値（speaker=3）にフォールバックする
    Given 環境変数 VOX_SPEAKER=not-a-number が設定されている
    When "vox-actor act --dry-run <path>" を実行する
    Then 終了コード 0 で完了する
    And "playback completed" 行に "speaker=3" が含まれる
    # test: test/e2e/act_comm_test.go::TestActE2E_Env_VOXSpeakerInvalid_DefaultFallback

  # ===== シグナル処理 =====

  Scenario: VOICEVOX 処理中に SIGINT を送ると 5 秒以内にプロセスが終了する
    Given ハングするスタブ VOICEVOX サーバーを起動する
    And "vox-actor act --engine-url <stub_url> <path>" をバックグラウンドで起動する
    When 150ms 後に SIGINT を送る
    Then 5 秒以内にプロセスが終了する
    # test: test/e2e/act_comm_test.go::TestActE2E_SIGINT_DuringVoicevox_Exits
