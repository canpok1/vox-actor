Feature: 各種設定の localStorage 永続化

  Scenario: 履歴上限・表示トグルが localStorage に保存され、リロード後も復元される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 履歴上限を 50 に変更する
    And 「キャラ名」チェックボックスをオフにする
    And 「スタイル」チェックボックスをオフにする
    And 「時刻」チェックボックスをオフにする
    Then localStorage に historySize=50 が保存される
    And localStorage に showSpeakerName=false が保存される
    And localStorage に showStyleName=false が保存される
    And localStorage に showTimestamp=false が保存される
    When ページをリロードする
    Then 履歴上限が 50 のままである
    And 「キャラ名」チェックボックスがオフのままである
    And 「スタイル」チェックボックスがオフのままである
    And 「時刻」チェックボックスがオフのままである
    # test: frontend/test/e2e/persistence.spec.ts::"履歴上限・表示トグルが localStorage に保存され、リロード後も復元される"
