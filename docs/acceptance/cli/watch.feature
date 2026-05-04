Feature: vox-actor watch コマンド

  # ===== Group A: 基本的なファイル監視とファイル後処理 =====

  Scenario: ドライランでファイルを検知すると再生完了後に done/ へ移動する
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .jsonl ファイルを書き込む
    Then "playback completed" ログが "[dry run]" を含む行として出力される
    And ファイルが done/ サブディレクトリに移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DryRun_DoneMove

  Scenario: --delete フラグ使用時は処理済みファイルを削除し done/ は作成しない
    Given テンポラリディレクトリで watch を --dry-run --delete で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .jsonl ファイルを書き込む
    Then "playback completed" ログが出力される
    And ファイルが削除される
    And done/ ディレクトリは作成されない
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DryRun_DeleteMode

  Scenario: --queue フラグで VOX_ACTOR_WORKSPACE/queue ディレクトリが自動作成され処理される
    Given 環境変数 VOX_ACTOR_WORKSPACE にワークスペースを設定して --queue --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    Then queue ディレクトリが自動作成されている
    When queue ディレクトリに .jsonl ファイルを書き込む
    Then "playback completed" ログが出力される
    And ファイルが done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_Queue_AutoCreatesAndProcesses

  # ===== Group A: 引数バリデーション =====

  Scenario Outline: 不正な引数を渡すと終了コード 2 と stderr が返る
    When <コマンド> を実行する
    Then 終了コード 2 で完了する
    And stderr が空でない

    Examples:
      | コマンド                                                    |
      | watch --queue --dry-run <ディレクトリパス>（位置引数あり） |
      | watch --stream <dir>（廃止済みフラグ）                      |
      | watch --stream-addr 0.0.0.0:9090 <dir>（廃止済みフラグ）   |
    # test: test/e2e/watch_test.go::TestWatchE2E_UsageError_ExitCode2

  Scenario: 引数なしで実行すると非ゼロ終了コードが返る
    When "vox-actor watch" を引数なしで実行する
    Then 終了コードが非ゼロである
    # test: test/e2e/watch_test.go::TestWatchE2E_NoArgs_NonZero

  # ===== Group B: ファイル形式ごとの処理 =====

  Scenario: .txt ファイルはファイル全体が1スクリプトとして処理される
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに複数行の .txt ファイルを書き込む
    Then "playback completed" が1回だけ出力される
    And ファイルが done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_TxtFile_DryRun_SingleScript

  Scenario: .json ファイルの各パラメータがプレイバックログに反映される
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When パラメータ付き .json ファイルをディレクトリに書き込む
    Then "playback completed" の行に "text=こんにちは", "speaker=11", "speed=1.2", "pitch=0.1", "intonation=1.5" が含まれる
    And ファイルが done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_JsonFile_DryRun_OverrideParams

  Scenario: .jsonl ファイルの各行が独立したスクリプトとして順番に処理される
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When 3行の .jsonl ファイルをディレクトリに書き込む
    Then "playback completed" が3回出力される
    And テキストの出力順が "いちぎょうめ → にぎょうめ → さんぎょうめ" である
    And ファイルが done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_JsonlFile_DryRun_MultipleScripts

  Scenario: 不正な JSON ファイルはエラーログを出力して done/ に移動し、監視を継続する
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When 不正 JSON ファイルをディレクトリに書き込む
    Then "read error" ログが出力される
    And 不正ファイルが done/ に移動する
    When 続けて有効な .txt ファイルをディレクトリに書き込む
    Then "playback completed" ログが出力される
    And 有効ファイルも done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_InvalidJson_ErrorThenContinues

  Scenario: サポート外の拡張子（.md）はプレーンテキストとして処理される
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When .md ファイルをディレクトリに書き込む
    Then "playback completed" ログが出力される
    And ファイルが done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_UnsupportedExtension_ProcessedAsText

  Scenario: 空ファイルは "playback completed" を出力せずに done/ に移動する
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When 空の .txt ファイルをディレクトリに書き込む
    Then ファイルが done/ に移動する
    And stderr に "playback completed" は含まれない
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_EmptyFile_MovedToDone

  # ===== Group C: 複数ディレクトリ並列監視 =====

  Scenario: 複数ディレクトリを指定すると各ディレクトリの "watching directory" ログが出力される
    Given 2つのテンポラリディレクトリで watch を --dry-run で起動する
    Then "watching directory" ログが2回出力される
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_MultipleDir_WatchingLogs

  Scenario: 複数ディレクトリに書き込んだファイルが両方処理される
    Given 2つのテンポラリディレクトリで watch を --dry-run で起動する
    And 両ディレクトリの "watching directory" ログが出るまで待機する
    When 各ディレクトリに .txt ファイルを書き込む
    Then "playback completed" が2回出力される
    And 各ファイルがそれぞれの done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_MultipleDir_BothProcessed

  Scenario: 複数ディレクトリを監視すると各ディレクトリに done/ が作成される
    Given 2つのテンポラリディレクトリで watch を --dry-run で起動する
    And 両ディレクトリの "watching directory" ログが出るまで待機する
    When 各ディレクトリに .txt ファイルを書き込む
    Then 各ファイルがそれぞれのディレクトリの done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_MultipleDir_EachDoneCreated

  # ===== Group D: 引数バリデーション（パス系） =====

  Scenario: ファイルパス（ディレクトリでない）を指定すると終了コード 2 が返る
    Given 空のファイルを作成する
    When "vox-actor watch --dry-run <file_path>" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_FilePath_ExitCode2

  Scenario: 存在しないパスを指定すると終了コード 2 が返る
    When "vox-actor watch --dry-run /nonexistent/path/12345" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_NonExistentPath_ExitCode2

  Scenario: 複数パスのうち1つが存在しない場合、終了コード 2 が返る
    Given テンポラリディレクトリと存在しないパスを用意する
    When "vox-actor watch --dry-run <valid_dir> /nonexistent/path/12345" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_MultiplePathsOneMissing_ExitCode2

  # ===== Group E: フラグと環境変数の優先順位 =====

  Scenario: --speaker フラグが VOX_SPEAKER 環境変数より優先される
    Given 環境変数 VOX_SPEAKER=10 が設定されている
    And テンポラリディレクトリで watch を --dry-run --speaker 5 で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then "playback completed" 行に "speaker=5" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_Flag_SpeakerOverridesEnv

  Scenario: --engine-url フラグが VOX_ENGINE_URL 環境変数より優先され、5xx サーバーに接続して非ゼロ終了する
    Given 環境変数 VOX_ENGINE_URL に到達不能アドレスを設定する
    And --engine-url に 5xx を返すスタブサーバーの URL を指定して watch を起動する
    Then watch プロセスが 10 秒以内に終了する
    And 終了コードが非ゼロである
    And stderr に "500" または "audio" が含まれる
    # test: test/e2e/watch_test.go::TestWatchE2E_Flag_EngineURLOverridesEnv

  Scenario: VOX_SPEAKER に不正な値を設定するとデフォルト値（speaker=3）にフォールバックする
    Given 環境変数 VOX_SPEAKER=not-a-number が設定されている
    And テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then "playback completed" 行に "speaker=3" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_Env_VOXSpeakerInvalid_DefaultFallback

  Scenario: .jsonl 各行パラメータがプレイバックログに反映される（ドライラン）
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When パラメータ付き .jsonl ファイルをディレクトリに書き込む
    Then "playback completed" 行に "speaker=11", "speed=1.2", "pitch=0.1", "intonation=1.5" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_JsonlPerLineOverride_DryRun

  # ===== Group F: ライフサイクル / シグナル =====

  Scenario: 監視中に SIGINT を送ると 5 秒以内に終了コード 0 で正常終了する
    Given テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When SIGINT を送る
    Then 5 秒以内にプロセスが終了する
    And 終了コードが 0 である
    # test: test/e2e/watch_test.go::TestWatchE2E_SIGINT_GracefulShutdown

  # ===== Group G: ヘルプ / 境界 =====

  Scenario: --help フラグで終了コード 0 かつ主要フラグが表示される
    When "vox-actor watch --help" を実行する
    Then 終了コード 0 で完了する
    And stdout に "--delete", "--queue", "--dry-run", "--speaker", "--engine-url" が含まれる
    # test: test/e2e/watch_test.go::TestWatchE2E_Help_ExitZero

  Scenario: done/ に同名ファイルが存在する場合、タイムスタンプ付きリネームで両ファイルが残る
    Given done/ ディレクトリに既存の "a.txt" を作成する
    And テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When 同名の "a.txt" をディレクトリに書き込む
    Then "playback completed" ログが出力される
    And 新しいファイルが done/ に移動する
    And done/ に少なくとも2つのファイルが存在する（既存 + タイムスタンプ付きリネーム）
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DoneCollision_TimestampRename

  Scenario: done/ ディレクトリ内のファイルは監視対象外である
    Given done/ ディレクトリに既存ファイルを作成する
    And テンポラリディレクトリで watch を --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When 2秒間待機する
    Then done/ 内の既存ファイルがそのまま残っている
    And stderr に "file detected" が含まれない
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DoneDir_NotMonitored

  # ===== Group H: ログ内容 =====

  Scenario: --verbose フラグで "watch mode starting" デバッグログが出力される
    Given テンポラリディレクトリで watch を --dry-run --verbose で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then "playback completed" ログが出力される
    And stderr に "watch mode starting" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_Verbose_DebugLogs

  Scenario: "watching directory" ログにディレクトリパスが含まれる
    Given テンポラリディレクトリで watch を --dry-run で起動する
    Then "watching directory" ログにディレクトリパスが含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_WatchingDirectoryLog

  Scenario: --verbose フラグで "file detected" ログにファイル名が含まれる
    Given テンポラリディレクトリで watch を --dry-run --verbose で起動する
    And "watching directory" ログが出るまで待機する
    When "detected.txt" をディレクトリに書き込む
    Then "file detected" ログに "detected.txt" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_FileDetectedLog

  # ===== Group I: --queue 系の追加検証 =====

  Scenario: --queue --delete の組み合わせで処理済みファイルが削除され done/ は作成されない
    Given 環境変数 VOX_ACTOR_WORKSPACE にワークスペースを設定して --queue --delete --dry-run で起動する
    And "watching directory" ログが出るまで待機する
    When queue ディレクトリに .txt ファイルを書き込む
    Then "playback completed" ログが出力される
    And ファイルが削除される
    And done/ ディレクトリは作成されない
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_Queue_Delete_Combination

  # ===== VOICEVOX 実通信エラー =====

  Scenario: ヘルスチェックで VOICEVOX が 5xx を返すと非ゼロ終了コードで終了する
    Given VOICEVOX のスタブサーバーが常に 500 を返す
    When watch を VOX_ENGINE_URL に向けて起動する
    Then watch プロセスが 10 秒以内に終了する
    And 終了コードが非ゼロである
    And stderr が空でない
    # test: test/e2e/watch_comm_test.go::TestWatchE2E_RealComm_HealthCheck5xx_ExitsNonZero

  Scenario: ヘルスチェックで VOICEVOX への接続が失敗すると非ゼロ終了コードで終了する
    Given 環境変数 VOX_ENGINE_URL=http://127.0.0.1:1 が設定されている
    When watch を起動する
    Then watch プロセスが 10 秒以内に終了する
    And 終了コードが非ゼロである
    # test: test/e2e/watch_comm_test.go::TestWatchE2E_RealComm_HealthCheckConnFailed_ExitsNonZero

  Scenario: ヘルスチェック成功後に音声クエリが失敗してもエラーを記録して監視を継続する
    Given ヘルスチェックは通過し /audio_query が 500 を返すスタブサーバーを用意する
    And 音声デバイスが存在する環境であること（skipIfNoAudioDevice）
    When watch を起動して "watching directory" ログが出るまで待機する
    And ファイルを書き込む
    Then "create query error" ログが出力される
    And ファイルが done/ に移動する
    When さらにもう1つファイルを書き込む
    Then 再び "create query error" ログが出力され、そのファイルも done/ に移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_comm_test.go::TestWatchE2E_RealComm_QueryFails_MonitoringContinues

  Scenario: /audio_query と /synthesis エンドポイントが呼ばれて合成パイプラインが実行される
    Given /audio_query と /synthesis が正常レスポンスを返すスタブサーバーを用意する
    And 音声デバイスが存在する環境であること（skipIfNoAudioDevice）
    When watch を起動して "watching directory" ログが出るまで待機する
    And ファイルを書き込む
    Then /audio_query が少なくとも1回呼ばれる
    And /synthesis が少なくとも1回呼ばれる
    And stderr に "create query error" や "synthesize error" が含まれない
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_comm_test.go::TestWatchE2E_RealComm_SynthPipeline_Invoked

  # ===== --viewer-url フォールバック禁止 =====

  Scenario: --viewer-url 明示時、viewer 接続失敗でローカル再生にフォールバックせず "silent play error" が記録される
    Given テンポラリディレクトリで watch を "--viewer-url http://127.0.0.1:1" で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then stderr に "silent play error" が含まれる
    And ファイルが done/ サブディレクトリに移動する
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_comm_test.go::TestWatchE2E_ViewerURL_ConnFailed_LogsError

  # ===== --pitch / --intonation 単独反映 =====

  Scenario: --pitch フラグ単独指定でプレイバックログに反映される（ドライラン）
    Given テンポラリディレクトリで watch を "--dry-run --pitch 0.05" で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then "playback completed" ログの行に "pitch=0.05" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DryRun_PitchFlag

  Scenario: --intonation フラグ単独指定でプレイバックログに反映される（ドライラン）
    Given テンポラリディレクトリで watch を "--dry-run --intonation 1.5" で起動する
    And "watching directory" ログが出るまで待機する
    When ディレクトリに .txt ファイルを書き込む
    Then "playback completed" ログの行に "intonation=1.5" が含まれる
    And SIGTERM 送信後に終了コード 0 で完了する
    # test: test/e2e/watch_test.go::TestWatchE2E_DryRun_IntonationFlag
