package app

import (
	"context"
	"log/slog"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// prefetchBufferSize はsynth pipelineのbuffered channelサイズ。
// 1なら「再生中に次1件だけを先行合成」する。
const prefetchBufferSize = 1

// synthStage は合成結果がどの段階まで進んだかを表す。
type synthStage int

const (
	synthStageOK synthStage = iota
	synthStageQueryFailed
	synthStageSynthesizeFailed
)

// synthResult は1セリフ分の合成結果。
type synthResult struct {
	script entity.Script
	// index は非空スクリプト中の1-basedインデックス。
	index int
	// wav はstageがsynthStageOKの場合のみ有効。
	wav   []byte
	stage synthStage
	// err はstage!=synthStageOKの場合の原因エラー。ラップはしない。
	err error
}

// synthPipelineConfig はstartSynthPipelineの設定。
type synthPipelineConfig struct {
	defaultSpeakerID  int
	defaultSpeed      *float64
	defaultPitch      *float64
	defaultIntonation *float64
	// synthesisLogLevel は "synthesis completed" ログの出力レベル。
	synthesisLogLevel slog.Level
}

// startSynthPipeline はプロデューサーgoroutineを起動し、非空スクリプトごとに
// CreateQuery -> Synthesize を順次実行し、結果をchannelで返す。
// バッファサイズ prefetchBufferSize により、再生中に次セリフ以降の合成を
// 先行実行して再生ギャップを隠す。
//
// channel は全スクリプト処理完了、またはctxキャンセル時にcloseされる。
// エラー時は stage 付きで result を送信し、caller が全体中断/スキップを判断する。
func startSynthPipeline(
	ctx context.Context,
	client VoicevoxClient,
	logger *slog.Logger,
	scripts []entity.Script,
	cfg synthPipelineConfig,
) <-chan synthResult {
	ch := make(chan synthResult, prefetchBufferSize)
	go func() {
		defer close(ch)
		index := 0
		for _, script := range scripts {
			if ctx.Err() != nil {
				return
			}
			if script.IsEmpty {
				continue
			}
			index++

			speakerID := script.ResolveSpeakerID(cfg.defaultSpeakerID)
			speed := script.ResolveSpeed(cfg.defaultSpeed)
			pitch := script.ResolvePitch(cfg.defaultPitch)
			intonation := script.ResolveIntonation(cfg.defaultIntonation)

			result := synthResult{script: script, index: index}

			query, err := client.CreateQuery(ctx, script.Text, speakerID)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				result.stage = synthStageQueryFailed
				result.err = err
				if !sendSynthResult(ctx, ch, result) {
					return
				}
				continue
			}
			logger.Debug("query created", "path", script.Path)

			q := query.WithOverrides(speed, pitch, intonation)
			wavData, err := client.Synthesize(ctx, &q, speakerID)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				result.stage = synthStageSynthesizeFailed
				result.err = err
				if !sendSynthResult(ctx, ch, result) {
					return
				}
				continue
			}
			logger.Log(ctx, cfg.synthesisLogLevel, "synthesis completed", "path", script.Path, "wavSize", len(wavData))

			result.stage = synthStageOK
			result.wav = wavData
			if !sendSynthResult(ctx, ch, result) {
				return
			}
		}
	}()
	return ch
}

// sendSynthResult はctxキャンセルを検知しつつchannelへ送信する。
// 正常に送信できた場合はtrue、ctxキャンセルでfalseを返す。
func sendSynthResult(ctx context.Context, ch chan<- synthResult, r synthResult) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}
