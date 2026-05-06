Feature: TimelineItem の個別 UI 要素

  # ── clip: メタ情報の表示切替 ──────────────────────────────────────

  Scenario: 「キャラ名」ON のとき話者名が表示される
    Given viewer フロントエンドが起動している
    When ページを開く（「キャラ名」チェックボックスはデフォルト ON）
    And clip イベントを受信する
    Then タイムラインアイテムに話者名が表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"「キャラ名」ON のとき話者名が表示される"

  Scenario: 「キャラ名」OFF のとき話者名が非表示になる
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「キャラ名」チェックボックスをオフにする
    And clip イベントを受信する
    Then タイムラインアイテムに話者名が表示されない
    # test: frontend/test/e2e/timeline-item.spec.ts::"「キャラ名」OFF のとき話者名が非表示になる"

  Scenario: 「スタイル」ON のときスタイル名が表示される
    Given viewer フロントエンドが起動している
    When ページを開く（「スタイル」チェックボックスはデフォルト ON）
    And clip イベントを受信する
    Then タイムラインアイテムにスタイル名が表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"「スタイル」ON のときスタイル名が表示される"

  Scenario: 「スタイル」OFF のときスタイル名が非表示になる
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「スタイル」チェックボックスをオフにする
    And clip イベントを受信する
    Then タイムラインアイテムにスタイル名が表示されない
    # test: frontend/test/e2e/timeline-item.spec.ts::"「スタイル」OFF のときスタイル名が非表示になる"

  Scenario: 「時刻」ON のときタイムスタンプが表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「時刻」チェックボックスをオンにする
    And clip イベントを受信する
    Then タイムラインアイテムに HH:MM:SS 形式の時刻が表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"「時刻」ON のときタイムスタンプが表示される"

  Scenario: 「時刻」OFF のときタイムスタンプが非表示になる
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「時刻」チェックボックスをオフにする
    And clip イベントを受信する
    Then タイムラインアイテムに HH:MM:SS 形式の時刻が表示されない
    # test: frontend/test/e2e/timeline-item.spec.ts::"「時刻」OFF のときタイムスタンプが非表示になる"

  # ── error: エラーカテゴリ別ラベル表示 ────────────────────────────

  Scenario: synthesis エラーは「合成エラー」ラベルが表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And error イベントを受信する（category=synthesis）
    Then タイムラインアイテムに「合成エラー」ラベルとメッセージが表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"synthesis エラーは「合成エラー」ラベルが表示される"

  Scenario: file エラーは「ファイルエラー」ラベルが表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And error イベントを受信する（category=file）
    Then タイムラインアイテムに「ファイルエラー」ラベルが表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"file エラーは「ファイルエラー」ラベルが表示される"

  Scenario: connection エラーは「接続エラー」ラベルが表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And error イベントを受信する（category=connection）
    Then タイムラインアイテムに「接続エラー」ラベルが表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"connection エラーは「接続エラー」ラベルが表示される"

  Scenario: error に speakerName と text が含まれている場合は表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And speakerName と text を含む error イベントを受信する
    Then タイムラインアイテムにテキストと話者名が表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"error に speakerName と text が含まれている場合は表示される"

  # ── Timeline: 履歴件数上限 ────────────────────────────────────────

  Scenario: 履歴件数が上限を超えると古いエントリが削除される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 履歴上限を 10 に設定する
    And URL が空の clip を 11 件受信する
    Then 最新10件はタイムラインに表示される
    And 最古のエントリはタイムラインから削除されている
    # test: frontend/test/e2e/timeline-item.spec.ts::"履歴件数が上限を超えると古いエントリが削除される"

  # ── TimelineItem: 再生ボタン ────────────────────────────────────

  Scenario: アイドル時は clip アイテムに「再生」ボタンが表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And clip イベントを受信する
    Then タイムラインアイテムに「再生」ボタンが表示される
    # test: frontend/test/e2e/timeline-item.spec.ts::"clip アイテムに「再生」ボタンが表示される"

  Scenario: 再生中はすべての clip アイテムの「再生」ボタンが非表示になる
    Given viewer フロントエンドが起動している
    When ページを開く
    And clip イベントを受信する
    And クリップが再生中である
    Then タイムラインアイテムに「再生」ボタンが表示されない
    # test: frontend/src/components/TimelineItem.test.tsx::"anyPlaying=true のとき onReplay が渡されていても「再生」ボタンが表示されない"

  Scenario: 再生が終わるとすべての clip アイテムの「再生」ボタンが再び表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And clip イベントを受信する
    And クリップが再生中である
    And 再生が終わりアイドル状態に戻る
    Then タイムラインアイテムに「再生」ボタンが表示される
    # test: frontend/src/components/TimelineItem.test.tsx::"anyPlaying=false のとき onReplay が渡されていれば「再生」ボタンが表示される"

  Scenario: 「再生」ボタン押下で POST /api/preview-clip に正しい clip パラメータが送信される
    Given viewer フロントエンドが起動している
    When ページを開く
    And text="再生テキスト", speakerId=3, speed=1.1 の clip イベントを受信する
    And タイムラインアイテムの「再生」ボタンをクリックする
    Then POST /api/preview-clip のリクエストが発生する
    And リクエストボディの text が "再生テキスト" である
    And リクエストボディの speaker_id が 3 である
    And リクエストボディの speed が 1.1 である
    # test: frontend/test/e2e/timeline-item.spec.ts::"「再生」ボタン押下で POST /api/preview-clip に正しい clip パラメータが送信される"
