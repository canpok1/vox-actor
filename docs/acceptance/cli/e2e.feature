Feature: vox-actor CLI 共通動作

  Scenario: "--version" フラグで "vox-actor version <文字列>" が1行だけ出力される
    When "vox-actor --version" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "vox-actor version <バージョン文字列>" の形式の行がちょうど1行含まれる
    # test: test/e2e/e2e_test.go::TestCLIVersion

  Scenario: "--help" フラグでサブコマンド一覧が表示される
    When "vox-actor --help" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "say"、"act"、"watch"、"config"、"audio-check" が含まれる
    # test: test/e2e/e2e_test.go::TestCLIHelp_ListsSubcommands

  Scenario: 引数なしで実行するとヘルプが表示される
    When "vox-actor" を引数なしで実行する
    Then 終了コード 0 で完了する
    And 標準出力に "vox-actor" と "Usage:" が含まれる
    # test: test/e2e/e2e_test.go::TestCLINoArgs_ShowsHelp

  Scenario: 存在しないサブコマンドを指定すると終了コードが 0 以外で stderr が出力される
    When "vox-actor unknown-cmd" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力が空でない
    # test: test/e2e/e2e_test.go::TestCLIUnknownSubcommand_NonZeroWithStderr

  Scenario: 存在しないフラグを指定すると終了コードが 0 以外になる
    When "vox-actor --no-such-flag" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/e2e_test.go::TestCLIUnknownFlag_NonZero
