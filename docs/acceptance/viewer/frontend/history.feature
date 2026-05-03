Feature: 配信タブ: 再生履歴

  Scenario: ページ読み込み時に /api/history の履歴がタイムラインに表示される
    Given viewer フロントエンドが起動している
    And /api/history に2件の履歴が登録されている
    When ページを開く
    Then SSE 接続ステータスが「接続中」になる
    And タイムラインに2件の履歴エントリが表示される
    # test: frontend/test/e2e/history.spec.ts::"ページ読み込み時に /api/history の履歴がタイムラインに表示される"

  Scenario: 履歴項目の表示後に audio の src が空のまま（音声再生が起きない）
    Given viewer フロントエンドが起動している
    And /api/history に1件の履歴が登録されている
    When ページを開く
    Then 履歴エントリがタイムラインに表示される
    And audio.src が空のまま（再生キューに積まれていない）
    # test: frontend/test/e2e/history.spec.ts::"履歴項目の表示後に audio の src が空のまま（音声再生が起きない）"

  Scenario: viewer 起動直後に履歴リストが末尾（最新エントリ）にスクロールされる
    Given viewer フロントエンドが起動している
    And /api/history に20件の履歴が登録されている
    When ページを開く
    Then 最新（最後）の履歴エントリがビューポート内に表示される
    And 最古（先頭）の履歴エントリはビューポート外になる
    # test: frontend/test/e2e/history.spec.ts::"viewer 起動直後に履歴リストが末尾（最新エントリ）にスクロールされる"

  Scenario: 履歴が空のとき従来どおりリストは空のまま
    Given viewer フロントエンドが起動している
    And /api/history の履歴が空である
    When ページを開く
    Then タイムラインリストが空のまま
    # test: frontend/test/e2e/history.spec.ts::"履歴が空のとき従来どおりリストは空のまま"

  Scenario: リロード後も最新の /api/history が表示される
    Given viewer フロントエンドが起動している
    And /api/history に1件の履歴が登録されている
    When ページを開く
    And /api/history に新たな履歴を1件追加する
    And ページをリロードする
    Then 追加した履歴エントリがタイムラインに表示される
    # test: frontend/test/e2e/history.spec.ts::"リロード後も最新の /api/history が表示される"
