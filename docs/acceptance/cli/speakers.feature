Feature: vox-actor speakers コマンド

  # ─────────────────────────────────────────
  # A. 親コマンド / ヘルプ / 引数バリデーション
  # ─────────────────────────────────────────

  Scenario: "speakers" のみ実行するとヘルプに "list" と "profile" が含まれる
    When "vox-actor speakers" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "speakers"、"list"、"profile" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_NoSubcmd_ShowsHelp

  Scenario: "speakers list --help" は終了コード 0 で完了する
    When "vox-actor speakers list --help" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_ListHelp_ExitZero

  Scenario: "speakers profile --help" は終了コード 0 で完了する
    When "vox-actor speakers profile --help" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_ProfileHelp_ExitZero

  Scenario: --id も --name も指定せずに "speakers profile" を実行するとエラーになる
    Given ワークスペースが設定されている
    When "vox-actor speakers profile" を実行する（フラグなし）
    Then 終了コードが 0 以外である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_ProfileNoFlags_ExitNonZero

  Scenario: --id と --name を同時に指定すると "speakers profile" がエラーになる
    Given ワークスペースが設定されている
    When "vox-actor speakers profile --id zundamon --name ずんだもん" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_ProfileBothFlags_ExitNonZero

  Scenario: "speakers list" に余分な位置引数を渡しても終了コード 0 で完了する
    Given ワークスペースとホームディレクトリが設定されている
    When "vox-actor speakers list extra-arg" を実行する
    Then 終了コード 0 で完了する
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_ListExtraArg_ExitZero

  # ─────────────────────────────────────────
  # B. 一覧・プロフィール表示（実ファイル経由）
  # ─────────────────────────────────────────

  Scenario: assets ディレクトリが空のとき "speakers list" は空の JSON 配列を返す
    Given ワークスペースとホームディレクトリが設定されている（assets なし）
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力が空の JSON 配列 "[]" である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_EmptyAssetsDir_ReturnsEmptyArray

  Scenario: 1件のスピーカーが存在するとき "speakers list" の出力にそのIDが含まれる
    Given プロジェクト assets に "zundamon" のスピーカーが存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "zundamon" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_OneSpeaker_InOutput

  Scenario: 複数スピーカーが存在するとき "speakers list" はID昇順でソートして返す
    Given プロジェクト assets に "zundamon"、"metan"、"kiritan" の3件が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON 配列が ID 昇順（kiritan → metan → zundamon）に並んでいる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_MultipleSpeakers_SortedByID

  Scenario: スピーカーの名前が "speakers list" の出力に反映される
    Given プロジェクト assets に名前 "ずんだもん" の "zundamon" が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "ずんだもん" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_SpeakerNameReflected

  Scenario: "--id" でプロフィールを取得するとすべてのフィールドが返る
    Given プロジェクト assets に profile 情報と description ファイルを持つ "zundamon" が存在する
    When "vox-actor speakers profile --id zundamon" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON に "id"、"name"、"pronoun"、"description" フィールドが含まれ正しい値を持つ
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_ByID_AllFields

  # ─────────────────────────────────────────
  # C. 階層マージ・優先順位
  # ─────────────────────────────────────────

  Scenario: プロジェクトスコープのみのスピーカーが "speakers list" に返る
    Given プロジェクト assets にのみ "zundamon" が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "zundamon" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_ProjectOnly_ReturnsSpeakers

  Scenario: ホームスコープのみのスピーカーが "speakers list" に返る
    Given ホーム assets にのみ "metan" が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "metan" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_HomeOnly_ReturnsSpeakers

  Scenario: プロジェクトとホームの両方にスピーカーがあるとき双方が "speakers list" に返る
    Given プロジェクト assets に "zundamon"、ホーム assets に "metan" が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "zundamon" と "metan" の両方が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_ProjectAndHome_ReturnsBoth

  Scenario: 同じIDのスピーカーがプロジェクトとホームにある場合、プロジェクト側が優先される（list）
    Given プロジェクト assets に名前 "プロジェクト版" の "shared"、ホーム assets に名前 "ホーム版" の "shared" が存在する
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON 配列が1件で名前が "プロジェクト版" である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_SameID_ProjectPriority

  Scenario: 同じIDのスピーカーがプロジェクトとホームにある場合、プロジェクト側が優先される（profile）
    Given プロジェクト assets に名前 "プロジェクト版" の "shared"、ホーム assets に名前 "ホーム版" の "shared" が存在する
    When "vox-actor speakers profile --id shared" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON の "name" が "プロジェクト版" である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_SameID_ProjectPriority

  # ─────────────────────────────────────────
  # D. profile --name 検索
  # ─────────────────────────────────────────

  Scenario: --name で一意のスピーカー名を検索するとプロフィールが返る
    Given プロジェクト assets に名前 "ずんだもん" の "zundamon" が存在する
    When "vox-actor speakers profile --name ずんだもん" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "zundamon" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_ByName_UniqueMatch_ReturnsProfile

  Scenario: --name で同名のスピーカーが複数マッチするとエラーになりIDが stderr に含まれる
    Given プロジェクト assets に同じ名前 "重複名前" を持つ "char1" と "char2" が存在する
    When "vox-actor speakers profile --name 重複名前" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力に "char1" または "char2" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_ByName_DuplicateName_ExitNonZero

  Scenario: --name で存在しないスピーカー名を検索するとエラーになる
    Given assets が空のワークスペースがある
    When "vox-actor speakers profile --name 存在しない名前" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_ByName_NotFound_ExitNonZero

  # ─────────────────────────────────────────
  # E. profile の description ファイル
  # ─────────────────────────────────────────

  Scenario: descriptionPath に指定されたファイルが存在する場合、その内容が "description" フィールドに返る
    Given プロジェクト assets に "descriptionPath: character.md" を持つ "zundamon" と実際のファイルが存在する
    When "vox-actor speakers profile --id zundamon" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON の "description" フィールドがファイルの内容と一致する
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_DescriptionFileExists_InOutput

  Scenario: descriptionPath に指定されたファイルが存在しない場合、"description" が空文字で警告が出る
    Given プロジェクト assets に "descriptionPath: nonexistent.md" を持つ "zundamon" が存在する（ファイルなし）
    When "vox-actor speakers profile --id zundamon" を実行する
    Then 終了コード 0 で完了する（警告付き）
    And 出力 JSON の "description" フィールドが空文字である
    And 標準エラー出力に "nonexistent.md" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_DescriptionFileMissing_EmptyDescriptionWithWarning

  # ─────────────────────────────────────────
  # F. 異常系・エラーハンドリング
  # ─────────────────────────────────────────

  Scenario: assets ディレクトリ自体が存在しない場合、"speakers list" は空リストを返す
    Given ワークスペースとホームディレクトリは設定されているが assets ディレクトリが存在しない
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON 配列が空である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_AssetsDirMissing_EmptyList

  Scenario: speaker.json が存在しないスピーカーディレクトリがある場合、警告を出してスキップする
    Given プロジェクト assets に "zundamon" ディレクトリが存在するが speaker.json がない
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON 配列が空である
    And 標準エラー出力に警告として "zundamon" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_SpeakerJSONMissing_WarningAndSkip

  Scenario: speaker.json が不正な JSON の場合、警告を出してスキップする
    Given プロジェクト assets の "zundamon" の speaker.json が "{invalid json}" である
    When "vox-actor speakers list" を実行する
    Then 終了コード 0 で完了する
    And 出力 JSON 配列が空である
    And 標準エラー出力に警告として "zundamon" が含まれる
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_List_SpeakerJSONInvalid_WarningAndSkip

  Scenario: 存在しない ID で "speakers profile" を実行するとエラーになる
    Given assets が空のワークスペースがある
    When "vox-actor speakers profile --id nonexistent" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/speakers_test.go::TestSpeakersE2E_Profile_ByID_NotFound_ExitNonZero
