Feature: vox-actor config コマンド

  Scenario: VOX_ACTOR_WORKSPACE 環境変数を設定して path.workspace を取得する
    Given 環境変数 "VOX_ACTOR_WORKSPACE" に一時ディレクトリが設定されている
    When "vox-actor config path.workspace" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が設定したワークスペースパスと一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathWorkspace_WithEnv

  Scenario: VOX_ACTOR_WORKSPACE 環境変数を設定して path.queue を取得する
    Given 環境変数 "VOX_ACTOR_WORKSPACE" に一時ディレクトリが設定されている
    When "vox-actor config path.queue" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<workspace>/queue" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathQueue_WithEnv

  Scenario: git リポジトリ内で VOX_ACTOR_WORKSPACE 未設定のとき path.workspace を取得する
    Given 一時 git リポジトリが初期化されている
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのリポジトリ内で "vox-actor config path.workspace" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<gitリポジトリルート>/.vox-actor" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathWorkspace_InsideGitRepo

  Scenario: git リポジトリ内で VOX_ACTOR_WORKSPACE 未設定のとき path.queue を取得する
    Given 一時 git リポジトリが初期化されている
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのリポジトリ内で "vox-actor config path.queue" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<gitリポジトリルート>/.vox-actor/queue" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathQueue_InsideGitRepo

  Scenario: VOX_ACTOR_WORKSPACE 環境変数を設定して path.tmp を取得する
    Given 環境変数 "VOX_ACTOR_WORKSPACE" に一時ディレクトリが設定されている
    When "vox-actor config path.tmp" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<workspace>/tmp" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathTmp_WithEnv

  Scenario: git リポジトリ内で VOX_ACTOR_WORKSPACE 未設定のとき path.tmp を取得する
    Given 一時 git リポジトリが初期化されている
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのリポジトリ内で "vox-actor config path.tmp" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<gitリポジトリルート>/.vox-actor/tmp" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathTmp_InsideGitRepo

  Scenario: git リポジトリ外で VOX_ACTOR_WORKSPACE 未設定のとき path.workspace を取得する
    Given git リポジトリ外の一時ディレクトリにいる
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのディレクトリ内で "vox-actor config path.workspace" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<カレントディレクトリ>/.vox-actor" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathWorkspace_OutsideGitRepo

  Scenario: git リポジトリ外で VOX_ACTOR_WORKSPACE 未設定のとき path.queue を取得する
    Given git リポジトリ外の一時ディレクトリにいる
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのディレクトリ内で "vox-actor config path.queue" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<カレントディレクトリ>/.vox-actor/queue" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathQueue_OutsideGitRepo

  Scenario: git リポジトリ外で VOX_ACTOR_WORKSPACE 未設定のとき path.tmp を取得する
    Given git リポジトリ外の一時ディレクトリにいる
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When そのディレクトリ内で "vox-actor config path.tmp" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が "<カレントディレクトリ>/.vox-actor/tmp" と一致する（1行＋改行）
    # test: test/e2e/config_test.go::TestConfigE2E_PathTmp_OutsideGitRepo

  Scenario: 未知のキーを指定すると終了コード 2 でエラーメッセージと有効キー一覧が出力される
    When "vox-actor config path.unknown" を実行する
    Then 終了コード 2 で完了する
    And 標準出力が空である
    And 標準エラー出力に "unknown key: path.unknown"、"path.queue"、"path.tmp"、"path.workspace" が含まれる
    # test: test/e2e/config_test.go::TestConfigE2E_UnknownKey_ExitCode2

  Scenario: PATH を空にして git が見つからない場合、終了コード 1 で stderr にエラーが出る
    Given PATH が空に設定されている
    And 環境変数 "VOX_ACTOR_WORKSPACE" が空文字に設定されている
    When git リポジトリ外のディレクトリで "vox-actor config path.workspace" を実行する
    Then 終了コード 1 で完了する
    And 標準出力が空である
    And 標準エラー出力が空でない
    # test: test/e2e/config_test.go::TestConfigE2E_GitNotFound_ExitCode1

  Scenario: 引数なしで "config" を実行すると終了コード 2 になる
    When "vox-actor config" を実行する（引数なし）
    Then 終了コード 2 で完了する
    # test: test/e2e/config_test.go::TestConfigE2E_NoArgs_ExitCode2

  Scenario: 引数が多すぎる場合に終了コード 2 になる
    When "vox-actor config path.queue extra" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/config_test.go::TestConfigE2E_TooManyArgs_ExitCode2

  Scenario: "config --help" は終了コード 0 で使用方法を表示する
    When "vox-actor config --help" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "config"、"key"、"Usage:" が含まれる
    # test: test/e2e/config_test.go::TestConfigE2E_Help_ExitZeroWithUsage
