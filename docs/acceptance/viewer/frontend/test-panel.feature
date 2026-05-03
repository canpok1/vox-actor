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

  Scenario: テスト再生ボタンで /test-clip にリクエストが飛び、audio.src も更新される
    Given viewer フロントエンドが起動している
    When ページを開く
    And 「音声テスト」タブに切り替える
    And 話者セレクタで話者ID=3 を選択する
    And 「テスト再生」ボタンをクリックする
    Then /test-clip?speaker=3 へのリクエストが発生する
    And audio.src が /test-clip?speaker=3 の URL になる
    # test: frontend/test/e2e/test-panel.spec.ts::"テスト再生ボタンで /test-clip にリクエストが飛び、audio.src も更新される"
