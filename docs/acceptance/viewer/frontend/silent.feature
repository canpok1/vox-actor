Feature: 無音モード

  Scenario: SilentBadge が表示され、音声テストタブは描画されない
    Given viewer フロントエンドが起動している
    And API ステータスが silent=true に設定されている
    When ページを開く
    Then 無音モードバッジが表示される
    And 「配信」タブが表示される
    And 「音声テスト」タブが表示されない
    # test: frontend/test/e2e/silent.spec.ts::"SilentBadge が表示され、音声テストタブは描画されない"

  Scenario: URL 空の clip イベントもタイムラインに表示される
    Given viewer フロントエンドが起動している
    And API ステータスが silent=true に設定されている
    When ページを開く
    And URL が空の clip イベントを受信する
    Then タイムラインに clip のテキストが表示される
    # test: frontend/test/e2e/silent.spec.ts::"URL 空の clip イベントもタイムラインに表示される"
