# レイヤー構成と責務ルール

このドキュメントは vox-actor のクリーンアーキテクチャにおけるレイヤー構成と、各層の責務・依存方向のルールを定義する。人間のレビュー基準であると同時に、`architecture-reviewer` サブエージェントの判断基準としても参照される。

## レイヤー構成

```
main.go (DI)
  └→ cmd/                CLI層: cobra コマンド定義、フラグ解析
       └→ internal/app/       アプリケーション層: ユースケース、ポート（インターフェース）定義
            └→ internal/domain/  ドメイン層: エンティティ、ビジネスルール（外部依存なし）

internal/infra/  インフラ層: ポートの具象実装（app 層のインターフェースを実装）
```

## 依存方向

- 許可される依存: `cmd → app → domain`、`infra → app → domain`
- 禁止される依存: 上記以外のすべての方向
  - 特に以下は禁止
    - `domain → app`、`domain → infra`、`domain → cmd`
    - `app → infra`、`app → cmd`
    - `infra → cmd`

依存方向の静的チェックは `depcheck.yml` によって import 文レベルで検証されている。`architecture-reviewer` は、その補完として import 文だけでは検出できない「ロジック配置の意味的な違反」を検出する。

## 各層の責務ルール

| 層 | 置くべきもの | 置いてはいけないもの |
|---|---|---|
| domain | エンティティ、ビジネスルール、値の変換・検証ロジック、ドメインが満たすべき不変条件 | 外部依存（HTTP、ファイル I/O、DB、ログ出力など）、フレームワーク依存 |
| app | ユースケースのオーケストレーション、ポート（インターフェース）定義、複数ポートの協調制御 | 具象実装への直接依存、HTTP リクエスト・ファイル操作などの副作用を持つ処理の直接記述 |
| infra | ポートの具象実装（HTTP 通信、ファイル操作、外部プロセス起動、リトライ・タイムアウト等） | ビジネスルール、ドメインエンティティのフィールドを条件分岐で書き換えるような処理 |
| cmd | CLI 引数・フラグの解析、依存の組み立て（DI）、シグナルハンドリング、標準入出力との橋渡し | ビジネスルール、直接的な外部通信、ユースケースの条件分岐 |

## よくある違反パターン

以下は過去の手動レビュー（例: #160）で発見された違反パターンや、クリーンアーキテクチャで典型的に発生する問題である。

### 1. ドメインロジックが infra 層に漏れている

**違反例:**

```go
// internal/infra/voicevox_client.go
func (c *VoicevoxClient) Synthesize(req *entity.SynthesisRequest) ([]byte, error) {
    // ❌ エンティティのフィールドを条件付きで書き換えている（ドメインロジック）
    if req.SpeedScale < 0.5 {
        req.SpeedScale = 0.5
    }
    if req.SpeedScale > 2.0 {
        req.SpeedScale = 2.0
    }
    // ... HTTP 通信 ...
}
```

**あるべき姿:** エンティティのメソッド（例: `SynthesisRequest.Normalize()` や `NewSynthesisRequest()` のコンストラクタ内）としてドメイン層に配置する。

### 2. app 層のインターフェースに infra の関心事が漏れている

**違反例:**

```go
// internal/app/port.go
type VoicevoxClient interface {
    // ❌ retryCount は infra の関心事（リトライ戦略）
    Synthesize(ctx context.Context, req *entity.SynthesisRequest, retryCount int) ([]byte, error)
}
```

**あるべき姿:** リトライ回数は infra 層の実装内部に閉じ込める（例: `RetryableVoicevoxClient` のコンストラクタで受け取る）。

### 3. cmd 層にビジネスロジックの条件分岐がある

**違反例:**

```go
// cmd/act.go
func runAct(cmd *cobra.Command, args []string) error {
    // ❌ ファイル拡張子による処理分岐はビジネスルール
    if strings.HasSuffix(args[0], ".json") {
        // JSON 用の処理
    } else if strings.HasSuffix(args[0], ".yaml") {
        // YAML 用の処理
    }
}
```

**あるべき姿:** ユースケース（app 層）またはドメイン層に処理分岐を移す。cmd 層は「どのユースケースを呼び出すか」の組み立てに専念する。

### 4. app 層から infra の具象型を直接 import している

**違反例:**

```go
// internal/app/say_usecase.go
import "github.com/canpok1/vox-actor/internal/infra"  // ❌ 具象パッケージへの依存

func (u *SayUsecase) Execute(...) error {
    client := infra.NewVoicevoxClient(...)  // ❌
}
```

**あるべき姿:** app 層はインターフェース（ポート）に依存し、具象は main.go / cmd 層で組み立てて注入する。

## アーキテクチャチェックの棲み分け

| 検証手段 | 対象 | 粒度 |
|---|---|---|
| `depcheck.yml` | import 文の方向 | 静的・機械的 |
| `architecture-reviewer` | ロジックの配置・責務境界 | 意味的・コンテキスト依存 |

両者は補完関係にあり、どちらかだけでは不十分である。
