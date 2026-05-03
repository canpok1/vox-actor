Feature: viewer 静的アセット・画像配信

  # ── フロントエンドハッシュ付きアセット（viewer_assets_test.go） ──────────────

  Background:
    Given viewer が起動してHTTPサーバーが待ち受け中である

  Scenario: ハッシュ付きフロントエンドアセットに 200 が返る
    Given index.html にハッシュ付きアセットパス（/assets/<hash>.js|css）が含まれる
    When そのアセットパスに GET リクエストを送る
    Then HTTP ステータス 200 が返る
    And レスポンスボディが空でない
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_Hashed_200

  Scenario: ハッシュ付きフロントエンドアセットに長期キャッシュヘッダーが付与される
    Given index.html にハッシュ付きアセットパスが含まれる
    When そのアセットパスに GET リクエストを送る
    Then レスポンスヘッダー Cache-Control が "public, max-age=31536000, immutable" である
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_Hashed_CacheControl

  Scenario: ハッシュ付き JS アセットの Content-Type に "javascript" が含まれる
    Given index.html に .js アセットパスが少なくとも1つ含まれる
    When その .js アセットパスに GET リクエストを送る
    Then レスポンスヘッダー Content-Type に "javascript" が含まれる
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_Hashed_ContentType_JS

  Scenario: ハッシュ付き CSS アセットの Content-Type に "text/css" が含まれる
    Given index.html に .css アセットパスが含まれる（ない場合はスキップ）
    When その .css アセットパスに GET リクエストを送る
    Then レスポンスヘッダー Content-Type に "text/css" が含まれる
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_Hashed_ContentType_CSS

  Scenario: 存在しないハッシュ付きアセットに 404 が返る
    When "/assets/nonexistent-abc12345.js" に GET リクエストを送る
    Then HTTP ステータス 404 が返る
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_NotFound_404

  Scenario: ハッシュ付きフロントエンドアセットとキャラクター画像が共存して両方配信される
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/face.png が配置されている
    When ハッシュ付きアセットパスに GET リクエストを送る
    Then HTTP ステータス 200 が返る
    When "/assets/images/zundamon/face.png" に GET リクエストを送る
    Then HTTP ステータス 200 が返る
    # test: test/e2e/viewer_assets_test.go::TestViewerE2E_Assets_RoutingMix

  # ── キャラクター画像（viewer_images_test.go） ────────────────────────────────

  Scenario: プロジェクト assets ディレクトリの画像を 200 で取得できる
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/face.png が配置されている
    When "/assets/images/zundamon/face.png" に GET リクエストを送る
    Then HTTP ステータス 200 が返る
    And レスポンスボディがファイルの内容と一致する
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_ProjectAssets_200

  Scenario: HOME assets ディレクトリの画像を 200 で取得できる
    Given "${HOME}/.vox-actor/assets/metan/face.png" が配置されている
    When "/assets/images/metan/face.png" に GET リクエストを送る
    Then HTTP ステータス 200 が返る
    And レスポンスボディがファイルの内容と一致する
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_HomeAssets_200

  Scenario: 同じキャラクターIDがプロジェクトとHOMEの両方にある場合プロジェクトが優先される
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/face.png にプロジェクト用コンテンツが配置されている
    And "${HOME}/.vox-actor/assets/zundamon/face.png" にHOME用コンテンツが配置されている
    When "/assets/images/zundamon/face.png" に GET リクエストを送る
    Then HTTP ステータス 200 が返る
    And レスポンスボディがプロジェクト側のコンテンツと一致する
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_ProjectPriorityOverHome

  Scenario: 存在しないキャラクターIDの画像に 404 が返る
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/face.png が配置されている
    When "/assets/images/nonexistent-char/face.png" に GET リクエストを送る
    Then HTTP ステータス 404 が返る
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_NotFound_404

  Scenario: パストラバーサル（../）を含む画像リクエストが拒否される
    Given VOX_ACTOR_WORKSPACE/assets/zundamon/face.png が配置されている
    When "/assets/images/zundamon/../../../etc/passwd" に GET リクエストを送る
    Then HTTP ステータスが 200 以外である
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_PathTraversal_Rejected

  Scenario: assets ディレクトリが存在しない場合に画像リクエストで 404 が返る
    Given プロジェクトにも HOME にも assets ディレクトリが存在しない
    When "/assets/images/zundamon/face.png" に GET リクエストを送る
    Then HTTP ステータス 404 が返る
    # test: test/e2e/viewer_images_test.go::TestViewerE2E_Image_NoAssetsDir_404
