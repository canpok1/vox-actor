Feature: 配信タブ: SSE と Timeline

  Scenario: SSE 接続後、clip イベントが Timeline に反映される
    Given viewer フロントエンドが起動している
    When ページを開く
    Then SSE 接続ステータスが「接続中」になる
    When clip イベントを受信する
    Then タイムラインに clip のテキストと話者名が表示される
    # test: frontend/test/e2e/stream.spec.ts::"SSE 接続後、clip イベントが Timeline に反映される"

  Scenario: error イベントが Timeline にエラーとして表示される
    Given viewer フロントエンドが起動している
    When ページを開く
    And error イベントを受信する（category=synthesis）
    Then タイムラインにエラーアイテムが表示される
    And エラーアイテムに「合成エラー」ラベルとメッセージが含まれる
    # test: frontend/test/e2e/stream.spec.ts::"error イベントが Timeline にエラーとして表示される"

  Scenario: SSE 切断後、自動再接続される
    Given viewer フロントエンドが起動している
    When ページを開く
    And SSE を強制切断する
    Then ステータスが「切断」になる
    And 再接続されステータスが「接続中」に戻る
    # test: frontend/test/e2e/stream.spec.ts::"SSE 切断後、自動再接続される"

  # ===== リングバッファ上限 / 複数ブラウザ =====

  Scenario: リングバッファ 20 件上限超過時に古いものから破棄される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 21 件の clip イベントを送信する
    Then タイムラインに最新 20 件のみ表示され、最初の 1 件は消えている
    # test: frontend/test/e2e/stream.spec.ts::"リングバッファ 20 件上限超過時に古いものから破棄される"

  Scenario: 複数ブラウザへの同時 SSE 配信
    Given viewer フロントエンドが起動している
    When 2 つのページを開く
    And clip イベントを送信する
    Then 両ページのタイムラインに clip のテキストが表示される
    # test: frontend/test/e2e/stream.spec.ts::"複数ブラウザへの同時 SSE 配信"
