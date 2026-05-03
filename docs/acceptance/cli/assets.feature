Feature: vox-actor assets コマンド

  # ─────────────────────────────────────────
  # A. 親コマンド / ヘルプ / 引数バリデーション
  # ─────────────────────────────────────────

  Scenario: "assets" のみ実行するとヘルプに "download" が含まれる
    When "vox-actor assets" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "assets" と "download" が含まれる
    # test: test/e2e/assets_test.go::TestAssetsE2E_NoSubcmd_ShowsHelp

  Scenario: "assets download --help" で "--speaker"、"--force"、"--scope" が表示される
    When "vox-actor assets download --help" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "--speaker"、"--force"、"--scope" が含まれる
    # test: test/e2e/assets_test.go::TestAssetsE2E_DownloadHelp_ExitZeroWithAllFlags

  Scenario: "assets download" に引数なしで実行すると終了コード 2 になる
    When "vox-actor assets download" を実行する（引数なし）
    Then 終了コード 2 で完了する
    # test: test/e2e/assets_test.go::TestAssetsE2E_DownloadNoArgs_ExitCode2

  Scenario: --scope に無効な値を指定すると終了コード 2 でスコープエラーが stderr に出る
    Given ワークスペースが設定されている
    When "vox-actor assets download --scope invalid file:///dummy" を実行する
    Then 終了コード 2 で完了する
    And 標準エラー出力にスコープエラーを示す文言が含まれる
    # test: test/e2e/assets_test.go::TestAssetsE2E_DownloadInvalidScope_ExitCode2

  # ─────────────────────────────────────────
  # B. ローカル fake git リポジトリでの clone 動作
  # ─────────────────────────────────────────

  Scenario: 通常の clone でスピーカーのアセットがワークスペースに配置される
    Given "zundamon" のアセット（icon.png、face.png）を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And ワークスペースの assets/zundamon/icon.png が存在する
    And ワークスペースの assets/zundamon/face.png が存在する
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_NormalClone

  Scenario: "--scope project" を指定すると assets がプロジェクトワークスペースに配置される
    Given "kiritan" のアセットを含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download --scope project <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And ワークスペースの assets/kiritan/icon.png が存在する
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_ScopeProject

  Scenario: "--scope home" を指定すると assets がホームディレクトリ配下に配置される
    Given "metan" のアセットを含む fake git リポジトリがある
    And HOME が設定されている
    When "vox-actor assets download --scope home <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And "$HOME/.vox-actor/assets/metan/icon.png" が存在する
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_ScopeHome

  Scenario: "--speaker" フラグを複数回指定すると指定したスピーカーのみダウンロードされる
    Given "zundamon"、"metan"、"kiritan" の3件を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download --speaker zundamon --speaker metan <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And assets/zundamon/icon.png と assets/metan/icon.png が存在する
    And assets/kiritan/icon.png は存在しない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_MultipleSpeakersRepeat

  Scenario: "--speaker" フラグにカンマ区切りで複数スピーカーを指定するとそれらのみダウンロードされる
    Given "zundamon"、"metan"、"kiritan" の3件を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download --speaker zundamon,metan <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And assets/zundamon/icon.png と assets/metan/icon.png が存在する
    And assets/kiritan/icon.png は存在しない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_MultipleSpeakersComma

  Scenario: "--force" フラグを付けると既存スピーカーを上書きして新しいファイルで置き換わる
    Given ワークスペースの assets/zundamon/ に古いファイル "old.png" が存在する
    And fake git リポジトリには "icon.png"（new-content）が登録されている
    When "vox-actor assets download --force <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And assets/zundamon/icon.png が存在する
    And assets/zundamon/old.png は存在しない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_Force_OverwritesExisting

  Scenario: "--force" なしで既に存在するスピーカーをダウンロードしようとするとエラーになる
    Given ワークスペースに assets/zundamon/ ディレクトリが既に存在する
    And fake git リポジトリに "zundamon" が登録されている
    When "vox-actor assets download <repoURL>" を実行する（--force なし）
    Then 終了コードが 0 以外である
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_NoForce_ExistingSpeaker_ExitNonZero

  # ─────────────────────────────────────────
  # C. エラー系
  # ─────────────────────────────────────────

  Scenario: 存在しないリポジトリ URL を指定するとエラーになり stderr が出力される
    Given ワークスペースが設定されている
    When "vox-actor assets download file:///nonexistent/path/to/repo" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力が空でない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_InvalidRepoURL_ExitNonZero

  Scenario: vox-actor-assets.json が存在しないリポジトリを指定するとエラーになる
    Given vox-actor-assets.json を含まない fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download <repoURL>" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力が空でない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_NoAssetsJSON_ExitNonZero

  Scenario: vox-actor-assets.json が不正な JSON の場合エラーになる
    Given "not-valid-json" という内容の vox-actor-assets.json を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download <repoURL>" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力が空でない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_InvalidJSON_ExitNonZero

  Scenario: vox-actor-assets.json に記載されたアセットパスが存在しない場合エラーになる
    Given vox-actor-assets.json に "zundamon" のパスが記載されているがそのパスにファイルがない fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download <repoURL>" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力が空でない
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_MissingAssetFile_ExitNonZero

  # ─────────────────────────────────────────
  # D. ログ
  # ─────────────────────────────────────────

  Scenario: 正常なダウンロード時に進捗ログとして "cloning" と "copying" が stderr に出力される
    Given "zundamon" を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "vox-actor assets download <repoURL>" を実行する
    Then 終了コード 0 で完了する
    And 標準エラー出力に "cloning" と "copying" が含まれる
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_ProgressLogs

  Scenario: "--verbose" フラグを付けると通常より多くのログ行が stderr に出力される
    Given "zundamon" を含む fake git リポジトリがある
    And ワークスペースが設定されている
    When "--verbose" なしと "--verbose" ありでそれぞれ "vox-actor assets download <repoURL>" を実行する
    Then 両方とも終了コード 0 で完了する
    And "--verbose" ありのログ行数が "--verbose" なしより多い
    # test: test/e2e/assets_test.go::TestAssetsE2E_Download_VerboseFlag_MoreLogs
