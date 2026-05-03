Feature: 再生キュー

  Scenario: 連続して clip を受信した場合 Timeline の表示順は受信順になる
    Given viewer フロントエンドが起動している
    When ページを開く
    And 連続して3件の clip イベントを受信する
    Then タイムラインに3件すべてが表示される
    And 表示順が受信順（timestamp 昇順）になっている
    # test: frontend/test/e2e/playback-queue.spec.ts::"連続して clip を受信した場合 Timeline の表示順は受信順になる"

  Scenario: muted 状態でも Timeline には clip が積まれる
    Given viewer フロントエンドが起動している
    When ページを開く（デフォルトで消音 ON）
    And clip イベントを受信する
    Then audio.muted が true のまま
    And タイムラインに clip が表示される
    # test: frontend/test/e2e/playback-queue.spec.ts::"muted 状態でも Timeline には clip が積まれる"

  Scenario: URL が空の clip は Timeline には表示されるが audio.src は更新されない
    Given viewer フロントエンドが起動している
    When ページを開く
    And URL が空の clip イベントを受信する
    Then タイムラインに clip が表示される
    And audio.src が空のまま
    # test: frontend/test/e2e/playback-queue.spec.ts::"URL が空の clip は Timeline には表示されるが audio.src は更新されない"

  Scenario: audio.src に clip の URL が反映される（再生キューへの積み込み確認）
    Given viewer フロントエンドが起動している
    When ページを開く
    And 消音チェックボックスをオフにする
    And URL 付きの clip イベントを受信する
    Then audio.src に blob URL が設定される（先読みにより fetch 完了）
    # test: frontend/test/e2e/playback-queue.spec.ts::"audio.src に clip の URL が反映される（再生キューへの積み込み確認）"
