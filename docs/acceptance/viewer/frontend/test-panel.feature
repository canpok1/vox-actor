Feature: 音声テストタブ

  Scenario: 話者選択は localStorage に保存され、リロード後も復元される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「音声テスト」タブに切り替える
    And 話者セレクタで話者ID=1 を選択する
    Then localStorage に testSpeakerId=1 が保存される
    When ページをリロードする
    And 「音声テスト」タブに切り替える
    Then 話者セレクタが話者ID=1 のままである
    # test: frontend/test/e2e/test-panel.spec.ts::"話者選択は localStorage に保存され、リロード後も復元される"

  Scenario: テスト再生ボタンで POST /api/play に skip_history:true でリクエストが飛ぶ
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「音声テスト」タブに切り替える
    And 話者セレクタで話者ID=3 を選択する
    And 「テスト再生」ボタンをクリックする
    Then POST /api/play に skip_history=true のリクエストが発生する
    And リクエストボディの clips[0].speaker_id が 3 である
    # test: frontend/test/e2e/test-panel.spec.ts::"テスト再生ボタンで POST /api/play に skip_history:true でリクエストが飛ぶ"
