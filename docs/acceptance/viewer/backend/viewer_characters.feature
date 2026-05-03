Feature: viewer キャラクター一覧 API（GET /api/characters）

  Background:
    Given viewer が起動してHTTPサーバーが待ち受け中である

  Scenario: assets ディレクトリが存在しない場合は enabled=false で空リストが返る
    Given プロジェクトにも HOME にも assets ディレクトリが存在しない
    When GET /api/characters にリクエストする
    Then HTTP ステータス 200 が返る
    And レスポンス JSON の "enabled" が false である
    And レスポンス JSON の "characters" が空配列である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_NoAssetsDir

  Scenario: プロジェクト assets のみにキャラクターが存在する場合はそのキャラクターが返る
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/ に speaker.json と画像を配置する（speakerName: ずんだもん、スタイル: ノーマル）
    When GET /api/characters にリクエストする
    Then レスポンス JSON の "enabled" が true である
    And "characters" に1件のエントリが含まれる
    And そのエントリの "speakerName" が "ずんだもん" である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_ProjectAssetsOnly

  Scenario: HOME assets のみにキャラクターが存在する場合はそのキャラクターが返る
    Given "${HOME}/.vox-actor/assets/metan/" に speaker.json と画像を配置する（speakerName: 四国めたん、スタイル: ノーマル）
    When GET /api/characters にリクエストする
    Then レスポンス JSON の "enabled" が true である
    And "characters" に1件のエントリが含まれる
    And そのエントリの "speakerName" が "四国めたん" である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_HomeAssetsOnly

  Scenario: 同じキャラクターIDがプロジェクトとHOMEの両方にある場合プロジェクトが優先される
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/ に speakerName "ずんだもんproject" のキャラクターを配置する
    And "${HOME}/.vox-actor/assets/zundamon/" に speakerName "ずんだもんhome" のキャラクターを配置する
    When GET /api/characters にリクエストする
    Then レスポンス JSON の "enabled" が true である
    And "characters" が1件（プロジェクト側のみ）である
    And そのエントリの "speakerName" が "ずんだもんproject" である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_ProjectPriorityOverHome

  Scenario: 複数スタイルを持つキャラクターはスタイルごとにエントリが展開される
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/ に "ノーマル" と "あまあま" の2スタイルを持つキャラクターを配置する
    When GET /api/characters にリクエストする
    Then レスポンス JSON の "enabled" が true である
    And "characters" に2件のエントリが含まれる
    And エントリに "ノーマル" スタイルが含まれる
    And エントリに "あまあま" スタイルが含まれる
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_StylesExpanded

  Scenario: 不正な speaker.json を持つキャラクターはスキップされ有効なキャラクターは返る
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/ に有効なキャラクターを配置する
    And VOX_ACTOR_WORKSPACE/assets/invalid-char/speaker.json に不正なJSONを配置する
    When GET /api/characters にリクエストする
    Then レスポンス JSON の "enabled" が true である
    And "characters" に1件（有効なキャラクターのみ）が含まれる
    And そのエントリの "speakerName" が "ずんだもん" である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_InvalidSpeakerJSONSkipped

  Scenario: /api/characters のレスポンスの Content-Type が JSON UTF-8 である
    When GET /api/characters にリクエストする
    Then レスポンスヘッダー Content-Type が "application/json; charset=utf-8" である
    # test: test/e2e/viewer_characters_test.go::TestViewerE2E_Characters_ContentType
