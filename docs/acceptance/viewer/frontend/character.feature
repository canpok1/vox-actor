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

  Scenario: タイムライン履歴アイテムの「再生」ボタン押下でキャラクターがステージに表示される
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true で characters に1件登録されている
    And 「キャラ画像」チェックボックスが有効になっている
    And タイムラインに再生履歴アイテムが1件存在する
    When タイムライン履歴アイテムの「再生」ボタンを押す
    Then 再生中のみ該当キャラクターがステージに表示される
    And 再生終了後に一定時間経過するとキャラクターがステージから消える
    # test: frontend/src/hooks/useCharacterStage.test.ts::"preview 経由再生（timestamp あり）でのキャラクター表示"
    # note: 単体テストで担保。E2Eはキャラクターステージ表示の視覚的確認が必要なため手動検証

  Scenario: 「音量テスト」ボタン経由のプレビュー再生ではキャラクターがステージに表示されない
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true で characters に1件登録されている
    And 「キャラ画像」チェックボックスが有効になっている
    When 音声テストパネルの「音量テスト」ボタンを押す
    Then キャラクターステージに何も表示されない
    # note: 音量テストは固定テキストでキャラクター紐付けなし。手動検証
