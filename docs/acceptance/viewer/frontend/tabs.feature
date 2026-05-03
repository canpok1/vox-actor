Feature: タブ切替

  Scenario: 配信タブと音声テストタブを切り替えられる
    Given viewer フロントエンドが起動している
    When ページを開く
    Then 「配信」タブパネルが表示され「音声テスト」タブパネルが非表示である
    When 「音声テスト」タブをクリックする
    Then 「音声テスト」タブパネルが表示され「配信」タブパネルが非表示になる
    When 「配信」タブをクリックする
    Then 「配信」タブパネルが表示され「音声テスト」タブパネルが非表示になる
    # test: frontend/test/e2e/tabs.spec.ts::"配信タブと音声テストタブを切り替えられる"

  Scenario: 選択したタブはリロード後も復元される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「音声テスト」タブをクリックする
    Then localStorage に activeTab=test が保存される
    When ページをリロードする
    Then 「音声テスト」タブパネルが表示され「配信」タブパネルが非表示である
    # test: frontend/test/e2e/tabs.spec.ts::"選択したタブはリロード後も復元される"
