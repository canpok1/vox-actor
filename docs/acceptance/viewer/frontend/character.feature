Feature: キャラ画像表示（配信タブ統合）

  Scenario: 「キャラ」タブが存在しないこと
    Given viewer フロントエンドが起動している
    When ページを開く
    Then 「キャラ」タブが表示されない
    # test: frontend/test/e2e/character.spec.ts::"「キャラ」タブが存在しないこと"

  Scenario: charactersEnabled=true のとき「キャラ画像」チェックボックスが配信タブに表示される
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true で characters に1件登録されている
    When ページを開く
    Then 配信タブに「キャラ画像」チェックボックスが表示される
    # test: frontend/test/e2e/character.spec.ts::"charactersEnabled=true のとき「キャラ画像」チェックボックスが配信タブに表示される"

  Scenario: charactersEnabled=false のとき「キャラ画像」チェックボックスが非表示になる
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=false である
    When ページを開く
    Then 「キャラ画像」チェックボックスが表示されない
    # test: frontend/test/e2e/character.spec.ts::"charactersEnabled=false のとき「キャラ画像」チェックボックスが非表示になる"
