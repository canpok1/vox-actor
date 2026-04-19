package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra/logging"
)

func TestHumanHandler_Handle_InfoMessageOnly(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Info("hello world")

	output := buf.String()
	// 先頭は空白2文字（Info記号）
	if !strings.HasPrefix(output, "  [") {
		t.Errorf("expected prefix '  [', got: %q", output)
	}
	// メッセージが含まれる
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected output to contain 'hello world', got: %q", output)
	}
	// 属性なしなので括弧がない
	if strings.Contains(output, "(") {
		t.Errorf("expected no attributes section, got: %q", output)
	}
}

func TestHumanHandler_Handle_InfoWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Info("scripts loaded", "path", "/tmp/test.txt", "count", 5)

	output := buf.String()
	if !strings.Contains(output, "scripts loaded (path=/tmp/test.txt, count=5)") {
		t.Errorf("expected formatted attrs, got: %q", output)
	}
}

func TestHumanHandler_Handle_WarnPrefix(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelWarn,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Warn("something wrong")

	output := buf.String()
	if !strings.HasPrefix(output, "! [") {
		t.Errorf("expected Warn prefix '! [', got: %q", output)
	}
}

func TestHumanHandler_Handle_ErrorPrefix(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelError,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Error("fatal error")

	output := buf.String()
	if !strings.HasPrefix(output, "x [") {
		t.Errorf("expected Error prefix 'x [', got: %q", output)
	}
}

func TestHumanHandler_Handle_DebugPrefix(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelDebug,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Debug("debug info")

	output := buf.String()
	if !strings.HasPrefix(output, "  [") {
		t.Errorf("expected Debug prefix '  [', got: %q", output)
	}
	if !strings.Contains(output, "debug info") {
		t.Errorf("expected output to contain 'debug info', got: %q", output)
	}
}

func TestHumanHandler_Enabled_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug to be disabled when level is Info")
	}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be enabled when level is Info")
	}
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Warn to be enabled when level is Info")
	}
}

func TestHumanHandler_Handle_DateTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Info("test")

	output := buf.String()
	// 日時フォーマット: [YYYY-MM-DD HH:MM:SS]
	// 正規表現的に確認: "  [2026-04-11 " のようなパターン
	if !strings.Contains(output, "[20") {
		t.Errorf("expected datetime in output, got: %q", output)
	}
	if strings.Contains(output, "+") {
		t.Errorf("expected no timezone offset in output, got: %q", output)
	}
}

func TestHumanHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})
	logger := slog.New(h).With("component", "test")

	logger.Info("hello")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("expected WithAttrs to include 'component=test', got: %q", output)
	}
}

func TestHumanHandler_Handle_ColorDisabledByNoColor(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelWarn,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Warn("warning")

	output := buf.String()
	// ANSI escape sequence should not be present
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI escape when NoColor=true, got: %q", output)
	}
}

func TestHumanHandler_Handle_ColorEnabledForWarn(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelWarn,
		NoColor: false,
	})
	logger := slog.New(h)

	logger.Warn("warning")

	output := buf.String()
	// Yellow ANSI escape
	if !strings.Contains(output, "\033[33m") {
		t.Errorf("expected yellow ANSI escape for Warn, got: %q", output)
	}
}

func TestHumanHandler_Handle_ColorEnabledForError(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelError,
		NoColor: false,
	})
	logger := slog.New(h)

	logger.Error("error msg")

	output := buf.String()
	// Red ANSI escape
	if !strings.Contains(output, "\033[31m") {
		t.Errorf("expected red ANSI escape for Error, got: %q", output)
	}
}

func TestHumanHandler_Handle_ColorEnabledForDebug(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelDebug,
		NoColor: false,
	})
	logger := slog.New(h)

	logger.Debug("debug msg")

	output := buf.String()
	// Gray ANSI escape (\033[90m)
	if !strings.Contains(output, "\033[90m") {
		t.Errorf("expected gray ANSI escape for Debug, got: %q", output)
	}
}

func TestHumanHandler_Handle_ColorEnabledForInfo(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: false,
	})
	logger := slog.New(h)

	logger.Info("info msg")

	output := buf.String()
	// Info should have no color (no ANSI escape)
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI escape for Info, got: %q", output)
	}
}

func TestHumanHandler_Handle_DryRunPrefix(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
		DryRun:  true,
	})
	logger := slog.New(h)

	logger.Info("playback completed", "path", "script.json", "text", "こんにちは")

	output := buf.String()
	// 時刻の直後に [dry run] プレフィックスが挿入される
	if !strings.Contains(output, "] [dry run] playback completed") {
		t.Errorf("expected '[dry run]' prefix after timestamp, got: %q", output)
	}
	// 属性も通常通り出力される
	if !strings.Contains(output, "(path=script.json, text=こんにちは)") {
		t.Errorf("expected attributes after message, got: %q", output)
	}
}

func TestHumanHandler_Handle_DryRunPrefix_Disabled(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
		DryRun:  false,
	})
	logger := slog.New(h)

	logger.Info("normal message")

	output := buf.String()
	if strings.Contains(output, "[dry run]") {
		t.Errorf("expected no '[dry run]' prefix when DryRun=false, got: %q", output)
	}
}

func TestHumanHandler_Handle_NewlineTerminated(t *testing.T) {
	var buf bytes.Buffer
	h := logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
	})
	logger := slog.New(h)

	logger.Info("test")

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected output to end with newline, got: %q", output)
	}
}
