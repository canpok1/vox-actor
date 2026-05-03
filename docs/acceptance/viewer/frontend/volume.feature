Feature: 音量・消音コントロール

  Scenario: 音量スライダーの値が localStorage に保存され、リロード後も復元される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 音量スライダーを 23 に設定する
    Then localStorage に volume=23 が保存される
    When ページをリロードする
    Then 音量スライダーの値が 23 のままである
    # test: frontend/test/e2e/volume.spec.ts::"音量スライダーの値が localStorage に保存され、リロード後も復元される"

  Scenario: 消音チェックボックスの操作で audio.muted が連動する
    Given viewer フロントエンドが起動している
    When ページを開く（デフォルトで audio.muted=true）
    Then audio.muted が true である
    When 消音チェックボックスをオフにする
    Then audio.muted が false になる
    When 消音チェックボックスをオンにする
    Then audio.muted が true に戻る
    # test: frontend/test/e2e/volume.spec.ts::"消音チェックボックスの操作で audio.muted が連動する"
