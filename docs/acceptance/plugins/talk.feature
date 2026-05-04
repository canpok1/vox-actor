Feature: vox-actor-plugin:talk スキルの play-script.sh

  # ─────────────────────────────────────────
  # A. 前提条件エラー
  # ─────────────────────────────────────────

  Scenario: vox-actor が PATH に無い場合に [ERROR] を stderr に出力して非ゼロ終了する
    Given vox-actor コマンドが PATH に存在しない環境
    And 有効な JSONL ファイルが存在する
    When "play-script.sh <jsonl_path>" を実行する
    Then 終了コードが非ゼロである
    And 標準エラー出力に "[ERROR]" が含まれる
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_NoVoxActor_ErrorExit

  Scenario: JSONL ファイルのパスが指定されない場合に [ERROR] を出力して非ゼロ終了する
    Given vox-actor コマンドが PATH に存在する
    When "play-script.sh" を引数なしで実行する
    Then 終了コードが非ゼロである
    And 標準出力に "[ERROR]" が含まれる
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_NoArgs_ErrorExit

  Scenario: 存在しない JSONL ファイルのパスを指定した場合に [ERROR] を出力して非ゼロ終了する
    Given vox-actor コマンドが PATH に存在する
    When "play-script.sh /nonexistent/path.jsonl" を実行する
    Then 終了コードが非ゼロである
    And 標準出力に "[ERROR]" が含まれる
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_NoFile_ErrorExit

  # ─────────────────────────────────────────
  # B. モード切替（direct / file）
  # ─────────────────────────────────────────

  Scenario: audio-check が成功する環境では direct モードで vox-actor act を呼び出す
    Given vox-actor コマンドが PATH に存在する
    And "vox-actor audio-check" が成功する環境である
    And 有効な JSONL ファイルが存在する
    When "play-script.sh <jsonl_path>" を実行する
    Then 終了コード 0 で完了する
    And "vox-actor act" が1回呼び出される
    And JSONL ファイルが削除される
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_AudioCheckOK_DirectMode

  Scenario: viewer-check が成功する環境では direct モードで vox-actor act を呼び出す
    Given vox-actor コマンドが PATH に存在する
    And "vox-actor audio-check" が失敗し "vox-actor viewer-check" が成功する環境である
    And 有効な JSONL ファイルが存在する
    When "play-script.sh <jsonl_path>" を実行する
    Then 終了コード 0 で完了する
    And "vox-actor act" が1回呼び出される
    And JSONL ファイルが削除される
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_ViewerCheckOK_DirectMode

  Scenario: audio-check も viewer-check も失敗する環境では file モードでキューに移動する
    Given vox-actor コマンドが PATH に存在する
    And "vox-actor audio-check" も "vox-actor viewer-check" も失敗する環境である
    And 有効な JSONL ファイルが存在する
    When "play-script.sh <jsonl_path>" を実行する
    Then 終了コード 0 で完了する
    And JSONL ファイルがキューディレクトリに移動している
    And "vox-actor act" は呼び出されない
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_NoDevice_FileMode

  # ─────────────────────────────────────────
  # C. エラーログのローテーション
  # ─────────────────────────────────────────

  Scenario: act 失敗時に play-script-errors.log が 200 行を超えると末尾 200 行に切り詰められる
    Given vox-actor コマンドが PATH に存在する
    And "vox-actor audio-check" が成功する環境である
    And play-script-errors.log に 201 行のログが既に存在する
    And "vox-actor act" が失敗する設定である
    And 有効な JSONL ファイルが存在する
    When "play-script.sh <jsonl_path>" を実行する
    Then play-script-errors.log の行数が 200 行以内である
    # test: test/e2e/play_script_test.go::TestPlayScriptE2E_LogRotation
