Feature: LipSyncImage 表示

  Scenario: charactersEnabled=false のとき「キャラ画像」チェックボックスが表示されず画像エリアも非表示
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=false である
    When ページを開く
    Then 「キャラ画像」チェックボックスが表示されない
    And 口パク画像エリアが非表示になる
    # test: frontend/test/e2e/lip-sync-image.spec.ts::"charactersEnabled=false のとき「キャラ画像」チェックボックスが表示されず画像エリアも非表示"

  Scenario: charactersEnabled=true でキャラ画像チェックON・clip受信後に口パク画像が表示される
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true でキャラクターが1件登録されている
    When ページを開く
    And 消音チェックボックスをオフにする
    And 「キャラ画像」チェックボックスをオンにする
    And そのキャラクターに対応する clip イベントを受信する
    Then 口パク画像（mouth closed / mouth open）が DOM に存在する
    # test: frontend/test/e2e/lip-sync-image.spec.ts::"charactersEnabled=true でキャラ画像チェックON・clip受信後に口パク画像が表示される"

  Scenario: 「キャラ画像」チェックを OFF にするとキャラ画像エリアが非表示になる
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true である
    When ページを開く
    And 「キャラ画像」チェックボックスをオフにする
    Then 口パク画像エリアが非表示になる
    # test: frontend/test/e2e/lip-sync-image.spec.ts::"「キャラ画像」チェックを OFF にするとキャラ画像エリアが非表示になる"

  Scenario: キャラに対応しない speakerName の clip 受信時は画像が表示されない
    Given viewer フロントエンドが起動している
    And API のキャラクター設定が enabled=true でキャラクターA が登録されている
    When ページを開く
    And 「キャラ画像」チェックボックスをオンにする
    And キャラクターA と一致しない話者の clip イベントを受信する
    Then タイムラインには clip が表示される
    And 口パク画像エリアは表示されない
    # test: frontend/test/e2e/lip-sync-image.spec.ts::"キャラに対応しない speakerName の clip 受信時は画像が表示されない"
