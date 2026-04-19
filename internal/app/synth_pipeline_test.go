package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

type pipelineSynthesizeArgs struct {
	query     *entity.AudioQuery
	speakerID int
}

type pipelineCreateQueryArgs struct {
	text      string
	speakerID int
}

// pipelineMockClient はsynth_pipelineテスト用のVoicevoxClientモック。
type pipelineMockClient struct {
	mu sync.Mutex

	// 呼び出しごとのエラーを制御する。nilならqueryを返す。
	createQueryErrs map[int]error
	synthesizeErrs  map[int]error

	// Synthesize遅延。prefetch競合を模擬したいケースで使う。
	synthesizeDelay time.Duration

	createQueryCalls []pipelineCreateQueryArgs
	synthesizeCalls  []pipelineSynthesizeArgs
}

func (m *pipelineMockClient) HealthCheck(_ context.Context) error { return nil }

func (m *pipelineMockClient) CreateQuery(_ context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	m.mu.Lock()
	callIndex := len(m.createQueryCalls)
	m.createQueryCalls = append(m.createQueryCalls, pipelineCreateQueryArgs{text: text, speakerID: speakerID})
	err := m.createQueryErrs[callIndex]
	m.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return &entity.AudioQuery{}, nil
}

func (m *pipelineMockClient) Synthesize(_ context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error) {
	m.mu.Lock()
	callIndex := len(m.synthesizeCalls)
	m.synthesizeCalls = append(m.synthesizeCalls, pipelineSynthesizeArgs{query: query, speakerID: speakerID})
	delay := m.synthesizeDelay
	err := m.synthesizeErrs[callIndex]
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	if err != nil {
		return nil, err
	}
	return []byte("wav"), nil
}

func TestStartSynthPipeline_ForwardsScriptsInOrder(t *testing.T) {
	client := &pipelineMockClient{}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
		{Path: "b.txt", Text: "B", IsEmpty: false},
		{Path: "c.txt", Text: "C", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(context.Background(), client, discardLogger(), scripts, cfg)

	var got []synthResult
	for r := range ch {
		got = append(got, r)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
	for i, want := range []string{"a.txt", "b.txt", "c.txt"} {
		if got[i].script.Path != want {
			t.Errorf("result[%d].script.Path = %q, want %q", i, got[i].script.Path, want)
		}
		if got[i].index != i+1 {
			t.Errorf("result[%d].index = %d, want %d", i, got[i].index, i+1)
		}
		if got[i].stage != synthStageOK {
			t.Errorf("result[%d].stage = %d, want OK", i, got[i].stage)
		}
		if len(got[i].wav) == 0 {
			t.Errorf("result[%d].wav is empty", i)
		}
	}
}

func TestStartSynthPipeline_SkipsEmptyScripts_IndexCountsNonEmptyOnly(t *testing.T) {
	client := &pipelineMockClient{}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
		{Path: "empty.txt", Text: "", IsEmpty: true},
		{Path: "b.txt", Text: "B", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(context.Background(), client, discardLogger(), scripts, cfg)

	var got []synthResult
	for r := range ch {
		got = append(got, r)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results (empty skipped), got %d", len(got))
	}
	if got[0].script.Path != "a.txt" || got[0].index != 1 {
		t.Errorf("first result: path=%q index=%d, want a.txt / 1", got[0].script.Path, got[0].index)
	}
	if got[1].script.Path != "b.txt" || got[1].index != 2 {
		t.Errorf("second result: path=%q index=%d, want b.txt / 2", got[1].script.Path, got[1].index)
	}
	if len(client.createQueryCalls) != 2 {
		t.Errorf("expected 2 CreateQuery calls (empty skipped), got %d", len(client.createQueryCalls))
	}
}

func TestStartSynthPipeline_PropagatesCreateQueryError(t *testing.T) {
	boom := errors.New("query boom")
	client := &pipelineMockClient{
		createQueryErrs: map[int]error{0: boom},
	}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
		{Path: "b.txt", Text: "B", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(context.Background(), client, discardLogger(), scripts, cfg)

	var got []synthResult
	for r := range ch {
		got = append(got, r)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results (error forwarded for first), got %d", len(got))
	}
	if got[0].stage != synthStageQueryFailed {
		t.Errorf("first stage = %d, want QueryFailed", got[0].stage)
	}
	if !errors.Is(got[0].err, boom) {
		t.Errorf("first err = %v, want %v", got[0].err, boom)
	}
	if got[1].stage != synthStageOK {
		t.Errorf("second stage = %d, want OK", got[1].stage)
	}
}

func TestStartSynthPipeline_PropagatesSynthesizeError(t *testing.T) {
	boom := errors.New("synth boom")
	client := &pipelineMockClient{
		synthesizeErrs: map[int]error{0: boom},
	}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(context.Background(), client, discardLogger(), scripts, cfg)

	got := drainPipeline(ch)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].stage != synthStageSynthesizeFailed {
		t.Errorf("stage = %d, want SynthesizeFailed", got[0].stage)
	}
	if !errors.Is(got[0].err, boom) {
		t.Errorf("err = %v, want %v", got[0].err, boom)
	}
}

func TestStartSynthPipeline_ContextCancel_ExitsProducerWithoutLeak(t *testing.T) {
	// 長尺synthを模擬し、consumer側が早期終了してもgoroutineがリークしないか確認する。
	ctx, cancel := context.WithCancel(context.Background())

	client := &pipelineMockClient{
		synthesizeDelay: 50 * time.Millisecond,
	}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
		{Path: "b.txt", Text: "B", IsEmpty: false},
		{Path: "c.txt", Text: "C", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	goroutinesBefore := runtime.NumGoroutine()

	ch := startSynthPipeline(ctx, client, discardLogger(), scripts, cfg)

	// 1件だけ受け取ってキャンセル
	<-ch
	cancel()

	// channelがcloseされるまで待つ（producer goroutineの終了検知）
	deadline := time.After(1 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				// channel closed -> producer exited
				goto done
			}
		case <-deadline:
			t.Fatal("producer goroutine did not exit within deadline")
		}
	}
done:
	// goroutineリーク検知のため少し待つ
	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	// 厳密比較は難しいがproducerが残っていれば +1 になるはず。寛容マージンを設ける。
	if goroutinesAfter > goroutinesBefore+1 {
		t.Errorf("possible goroutine leak: before=%d after=%d", goroutinesBefore, goroutinesAfter)
	}
}

func TestStartSynthPipeline_ContextAlreadyCancelled_ClosesChannelImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &pipelineMockClient{}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(ctx, client, discardLogger(), scripts, cfg)

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel closed (no result sent), got a result")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("producer did not close channel after ctx already cancelled")
	}

	if len(client.createQueryCalls) != 0 {
		t.Errorf("expected 0 CreateQuery calls when ctx pre-cancelled, got %d", len(client.createQueryCalls))
	}
}

func TestStartSynthPipeline_LogsSynthesisCompletedAtConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := &pipelineMockClient{}
	scripts := []entity.Script{
		{Path: "a.txt", Text: "A", IsEmpty: false},
	}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelInfo}

	ch := startSynthPipeline(context.Background(), client, logger, scripts, cfg)
	drainPipeline(ch)

	output := buf.String()
	if !strings.Contains(output, `level=INFO`) || !strings.Contains(output, "synthesis completed") {
		t.Errorf("expected INFO synthesis completed log, got: %s", output)
	}
	if !strings.Contains(output, `level=DEBUG`) || !strings.Contains(output, "query created") {
		t.Errorf("expected DEBUG query created log, got: %s", output)
	}
}

func TestStartSynthPipeline_MultipleInvocations_NoGoroutineLeak(t *testing.T) {
	// watchモードでファイルをまたいで複数回pipelineを起動するシナリオを模擬する。
	// 各呼び出しで正常に全スクリプトを処理すればgoroutineがリークしないことを確認する。
	goroutinesBefore := runtime.NumGoroutine()

	client := &pipelineMockClient{}
	cfg := synthPipelineConfig{defaultSpeakerID: 3, synthesisLogLevel: slog.LevelDebug}

	for range 20 {
		scripts := []entity.Script{
			{Path: "a.txt", Text: "A", IsEmpty: false},
			{Path: "b.txt", Text: "B", IsEmpty: false},
		}
		ch := startSynthPipeline(context.Background(), client, discardLogger(), scripts, cfg)
		drainPipeline(ch)
	}

	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	if goroutinesAfter > goroutinesBefore+2 {
		t.Errorf("possible goroutine leak across invocations: before=%d after=%d", goroutinesBefore, goroutinesAfter)
	}
}

func drainPipeline(ch <-chan synthResult) []synthResult {
	var results []synthResult
	for r := range ch {
		results = append(results, r)
	}
	return results
}
