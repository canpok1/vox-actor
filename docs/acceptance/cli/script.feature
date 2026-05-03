Feature: vox-actor script コマンド

  # ─────────────────────────────────────────
  # A. 基本的な書き出し挙動
  # ─────────────────────────────────────────

  Scenario: .jsonl ファイルを新規作成してテキストを書き込む
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl こんにちは" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの "text" フィールドが "こんにちは" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_JsonlCreate

  Scenario: .json ファイルを新規作成してテキストを書き込む
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.json テスト" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの "text" フィールドが "テスト" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_JsonCreate

  Scenario: .txt ファイルを新規作成してテキストを書き込む
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.txt テキスト本文" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの内容が "テキスト本文" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_TxtCreate

  Scenario: 同じ .jsonl ファイルに2回追記すると2行になる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl 1行目" を実行する
    And "vox-actor script append <dir>/out.jsonl 2行目" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルが2行あり、1行目の "text" が "1行目"、2行目の "text" が "2行目" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_Append

  Scenario: 拡張子なしのパスを指定するとエラーになる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/noext テスト" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力に "no extension" が含まれる
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_NoExtension

  Scenario: サポート外の拡張子（.md）を指定するとエラーになる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.md テスト" を実行する
    Then 終了コードが 0 以外である
    And 標準エラー出力に "unsupported" が含まれる
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_UnsupportedExtension

  Scenario: 存在しない親ディレクトリを指定するとエラーになる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/nonexistent/out.jsonl テスト" を実行する
    Then 終了コードが 0 以外である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Basic_NoParentDir

  # ─────────────────────────────────────────
  # B. ボイスパラメータの反映
  # ─────────────────────────────────────────

  Scenario: ボイスパラメータを指定するとすべて出力に反映される
    Given 一時ディレクトリが存在する
    When "vox-actor script append --speaker 5 --speed 1.2 --pitch 0.05 --intonation 1.5 <dir>/out.jsonl パラメータテスト" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの "speaker" が 5、"speed" が 1.2、"pitch" が 0.05、"intonation" が 1.5 である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_VoiceParams_Reflected

  Scenario: ボイスパラメータを省略すると出力 JSON に含まれない（omitempty）
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl 省略テスト" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルに "speaker"、"speed"、"pitch"、"intonation" フィールドが存在しない
    # test: test/e2e/script_test.go::TestScriptAppendE2E_VoiceParams_Omitempty

  Scenario: .txt 出力でボイスパラメータを指定すると警告ログが出る
    Given 一時ディレクトリが存在する
    When "vox-actor script append --speaker 3 <dir>/out.txt テキスト警告" を実行する
    Then 終了コード 0 で完了する
    And 標準エラー出力に "text-format output cannot store voice parameters" が含まれる
    # test: test/e2e/script_test.go::TestScriptAppendE2E_VoiceParams_TxtWarn

  # ─────────────────────────────────────────
  # C. ディレクトリ書き出し
  # ─────────────────────────────────────────

  Scenario: 末尾にパス区切り文字を付けた既存ディレクトリを指定すると *_say.json が生成される
    Given 既存の一時ディレクトリが存在する
    When "vox-actor script append <dir>/ ディレクトリ指定" を実行する（末尾スラッシュあり）
    Then 終了コード 0 で完了する
    And ディレクトリ内に *_say.json ファイルが1つ生成され、"text" が "ディレクトリ指定" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Dir_TrailingSlash

  Scenario: 末尾スラッシュなしで既存ディレクトリを指定すると *_say.json が生成される
    Given 既存の一時ディレクトリが存在する
    When "vox-actor script append <dir> 既存ディレクトリ" を実行する
    Then 終了コード 0 で完了する
    And ディレクトリ内に *_say.json ファイルが1つ生成される
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Dir_ExistingDir

  Scenario: 末尾スラッシュ付きの新規ディレクトリを指定するとディレクトリが作成されて *_say.json が生成される
    Given 存在しない新規サブディレクトリパスがある
    When "vox-actor script append <newdir>/ 新規ディレクトリ" を実行する
    Then 終了コード 0 で完了する
    And 新規ディレクトリが作成される
    And ディレクトリ内に *_say.json ファイルが1つ生成される
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Dir_CreateNew

  # ─────────────────────────────────────────
  # D. 引数バリデーション
  # ─────────────────────────────────────────

  Scenario Outline: 引数が不正なとき終了コード 2 を返す
    When "vox-actor <args>" を実行する
    Then 終了コード 2 で完了する

    Examples:
      | ケース名          | args                                      |
      | 引数なし          | script append                             |
      | ファイルのみ      | script append <dir>/out.jsonl             |
      | 引数が多すぎる    | script append <dir>/out.jsonl a b         |
    # test: test/e2e/script_test.go::TestScriptAppendE2E_ArgValidation_ExitCode2

  # ─────────────────────────────────────────
  # E. ヘルプ / 親コマンド / 環境変数
  # ─────────────────────────────────────────

  Scenario: "script" のみ実行するとヘルプに "append" が含まれる
    When "vox-actor script" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "append" が含まれる
    # test: test/e2e/script_test.go::TestScriptE2E_Help_ScriptAlone

  Scenario: "script append --help" でフラグ一覧が表示され、不要フラグが含まれない
    When "vox-actor script append --help" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "--speaker"、"--speed"、"--pitch"、"--intonation"、"--verbose" が含まれる
    And 標準出力に "--engine-url"、"--dry-run" が含まれない
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Help_Flags

  Scenario: 環境変数 VOX_SPEAKER のみ設定してもフラグ未指定なら speaker フィールドは省略される
    Given 環境変数 "VOX_SPEAKER=11" が設定されている
    When "vox-actor script append <dir>/out.jsonl 環境変数" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルに "speaker" フィールドが存在しない
    # test: test/e2e/script_test.go::TestScriptAppendE2E_EnvVar_VOXSpeaker

  # ─────────────────────────────────────────
  # F. ログ内容
  # ─────────────────────────────────────────

  Scenario: 正常完了時に "output completed" ログが出力され path 属性を含む
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl ログテスト" を実行する
    Then 終了コード 0 で完了する
    And 標準エラー出力に "output completed" を含む行があり、その行に "path=" が含まれる
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Log_OutputCompleted

  Scenario: --verbose フラグを付けると "output completed" ログが出力される
    Given 一時ディレクトリが存在する
    When "vox-actor script append --verbose <dir>/out.jsonl 詳細ログ" を実行する
    Then 終了コード 0 で完了する
    And 標準エラー出力に "output completed" が含まれる
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Log_Verbose

  # ─────────────────────────────────────────
  # G. 文字エンコーディング / 特殊文字
  # ─────────────────────────────────────────

  Scenario: マルチバイト文字・絵文字を含むテキストが正しく書き込まれる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl 日本語テスト🎉" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの "text" フィールドが "日本語テスト🎉" である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Encoding_Multibyte

  Scenario: ダブルクォートやバックスラッシュを含む文字列が正しく書き込まれる
    Given 一時ディレクトリが存在する
    When ダブルクォートとバックスラッシュを含むテキストで "vox-actor script append" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルの "text" フィールドが元のテキストと一致する
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Encoding_SpecialChars

  Scenario: 空文字列を指定すると "text" フィールドが空文字になる
    Given 一時ディレクトリが存在する
    When "vox-actor script append <dir>/out.jsonl 空文字" を実行する（テキストは空文字列）
    Then 終了コード 0 で完了する
    And 出力ファイルの "text" フィールドが空文字列である
    # test: test/e2e/script_test.go::TestScriptAppendE2E_Encoding_EmptyText

  # ─────────────────────────────────────────
  # script write E2E テスト
  # ─────────────────────────────────────────

  Scenario: --json フラグで複数エントリを一括書き込みできる
    Given 一時ディレクトリが存在する
    When "vox-actor script write --json '[{"text":"おはようなのだ","speaker":3,"intonation":1.1},{"text":"今日もがんばるのだ","speaker":3,"speed":1.0}]' <dir>/out.jsonl" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルが2行あり、1行目の "text" が "おはようなのだ"・"speaker" が 3・"intonation" が 1.1・"speed" が省略、2行目の "text" が "今日もがんばるのだ"・"speed" が 1.0 である
    # test: test/e2e/script_test.go::TestScriptWriteE2E_Basic

  Scenario: script write は既存ファイルを上書きする
    Given append で "古いセリフ" が書き込まれた .jsonl ファイルがある
    When "vox-actor script write --json '[{"text":"新しいセリフ"}]' <dir>/out.jsonl" を実行する
    Then 終了コード 0 で完了する
    And 出力ファイルが1行のみで "text" が "新しいセリフ" である
    # test: test/e2e/script_test.go::TestScriptWriteE2E_OverwritesExisting

  Scenario: --json に不正な JSON を渡すと終了コード 2 を返す
    Given 一時ディレクトリが存在する
    When "vox-actor script write --json not-json <dir>/out.jsonl" を実行する
    Then 終了コード 2 で完了する
    # test: test/e2e/script_test.go::TestScriptWriteE2E_InvalidJson_ExitCode2

  Scenario: "script" のみ実行するとヘルプに "write" が含まれる
    When "vox-actor script" を実行する
    Then 終了コード 0 で完了する
    And 標準出力に "write" が含まれる
    # test: test/e2e/script_test.go::TestScriptE2E_Help_ContainsWrite
