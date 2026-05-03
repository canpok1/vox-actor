Feature: viewer バックエンド 起動・ステータス・インデックス・viewer-check

  # ── viewer_test.go ──────────────────────────────────────────────

  Scenario: /api/status エンドポイントが JSON を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    Then 標準エラーに "stream server listening" が出力される
    And ログから addr フィールドが抽出できる
    When GET /api/status を送信する
    Then ステータスコード 200 が返る
    And レスポンスボディが JSON である
    And JSON に "silent" フィールドが含まれる
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_test.go::TestViewerE2E_StatusEndpoint

  Scenario Outline: 不正なポート番号を指定すると終了コード 2 が返る
    Given vox-actor バイナリがビルド済みである
    When "vox-actor viewer --port <port>" を実行する
    Then 終了コード 2 が返る
    And 標準エラーが空でない

    Examples:
      | port  |
      | 0     |
      | 99999 |
    # test: test/e2e/viewer_test.go::TestViewerE2E_InvalidPort_ReturnsUsageError

  Scenario: /api/history が起動後に追記されたエントリを返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    And 一時ディレクトリを HOME・VOX_ACTOR_WORKSPACE に設定している
    When "vox-actor viewer --port <port>" を起動する
    Then 標準エラーに "stream server listening" が出力される
    When viewer 起動後に今日の日付の JSONL 履歴ファイルにエントリを追記する
    And GET /api/history を送信する
    Then ステータスコード 200 が返る
    And レスポンスボディが JSON であり entries 配列が 1 件である
    And エントリの text が "起動後追記テキスト" である
    And SIGTERM 送信後に終了コード 0 で終了する
    # test: test/e2e/viewer_test.go::TestViewerE2E_APIHistory_ReturnsUpdatedHistory

  # ── viewer_check_test.go ─────────────────────────────────────────

  Scenario: viewer-check に予期しない引数を渡すと終了コード 2 が返る
    Given vox-actor バイナリがビルド済みである
    When "vox-actor viewer-check unexpected" を実行する
    Then 終了コード 2 が返る
    # test: test/e2e/viewer_check_test.go::TestViewerCheckE2E_UnexpectedArg_ExitCode2

  Scenario: viewer-check に未知のフラグを渡すと非ゼロで終了する
    Given vox-actor バイナリがビルド済みである
    When "vox-actor viewer-check --bogus" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/viewer_check_test.go::TestViewerCheckE2E_UnknownFlag_NonZeroExit

  Scenario: viewer-check --help がゼロ終了し使用方法を出力する
    Given vox-actor バイナリがビルド済みである
    When "vox-actor viewer-check --help" を実行する
    Then 終了コード 0 が返る
    And 標準出力に "viewer-check" が含まれる
    And 標準出力に "--verbose" が含まれる
    # test: test/e2e/viewer_check_test.go::TestViewerCheckE2E_Help_ExitZeroWithUsage

  Scenario: viewer が起動していない環境では viewer-check が非ゼロで終了する
    Given vox-actor バイナリがビルド済みである
    And viewer プロセスが起動していない
    When "vox-actor viewer-check" を実行する
    Then 終了コードが 0 以外である（viewer が起動中の環境では 0 になる場合がある）
    # test: test/e2e/viewer_check_test.go::TestViewerCheckE2E_NoViewer_NonZeroExit

  # ── viewer_index_test.go ─────────────────────────────────────────

  Scenario: GET / がステータスコード 200 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET / を送信する
    Then ステータスコード 200 が返る
    # test: test/e2e/viewer_index_test.go::TestViewerE2E_Index_StatusCode

  Scenario: GET / の Content-Type が text/html を含む
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET / を送信する
    Then レスポンスヘッダの Content-Type が "text/html" を含む
    # test: test/e2e/viewer_index_test.go::TestViewerE2E_Index_ContentType

  Scenario: index.html がハッシュ付きアセットを参照し各アセットが 200 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET / を送信する
    Then レスポンスボディに "/assets/<ハッシュ>..." 形式のアセットパスが 1 件以上含まれる
    And 各アセットパスへの GET がステータスコード 200 を返す
    # test: test/e2e/viewer_index_test.go::TestViewerE2E_Index_HashedAssets

  Scenario: index.html が必須文字列を含む
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET / を送信する
    Then レスポンスボディに "<!DOCTYPE html>" が含まれる
    And レスポンスボディに 'id="root"' が含まれる
    And レスポンスボディに "vox-actor" が含まれる
    # test: test/e2e/viewer_index_test.go::TestViewerE2E_Index_KeyStrings

  Scenario: 存在しないパスへの GET が 404 を返す
    Given vox-actor バイナリがビルド済みである
    And 空きポートを取得している
    When "vox-actor viewer --port <port>" を起動する
    And GET /unknown-path を送信する
    Then ステータスコード 404 が返る
    # test: test/e2e/viewer_index_test.go::TestViewerE2E_Index_UnknownPath
