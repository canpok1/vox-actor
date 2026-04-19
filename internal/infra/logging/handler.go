package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// HumanHandlerOptions は HumanHandler の設定オプション。
type HumanHandlerOptions struct {
	Level   slog.Level
	NoColor bool
	// DryRun を true にすると、時刻の直後に "[dry run] " プレフィックスが挟まれる。
	DryRun bool
}

// HumanHandler は人間が読みやすい形式でログを出力する slog.Handler 実装。
type HumanHandler struct {
	w      io.Writer
	opts   HumanHandlerOptions
	attrs  []slog.Attr
	groups []string
	mu     *sync.Mutex
}

// NewHumanHandler は新しい HumanHandler を生成する。
func NewHumanHandler(w io.Writer, opts *HumanHandlerOptions) *HumanHandler {
	o := HumanHandlerOptions{
		Level: slog.LevelInfo,
	}
	if opts != nil {
		o = *opts
	}
	return &HumanHandler{
		w:    w,
		opts: o,
		mu:   &sync.Mutex{},
	}
}

// Enabled は指定レベルのログを出力するかどうかを返す。
func (h *HumanHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle はログレコードをフォーマットして出力する。
func (h *HumanHandler) Handle(_ context.Context, r slog.Record) error {
	prefix := levelPrefix(r.Level)
	timeStr := r.Time.Format("2006-01-02 15:04:05")

	var attrs []slog.Attr
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	attrStr := ""
	if len(attrs) > 0 {
		attrStr = " ("
		for i, a := range attrs {
			if i > 0 {
				attrStr += ", "
			}
			attrStr += fmt.Sprintf("%s=%v", a.Key, a.Value)
		}
		attrStr += ")"
	}

	dryRunPrefix := ""
	if h.opts.DryRun {
		dryRunPrefix = "[dry run] "
	}
	line := fmt.Sprintf("%s[%s] %s%s%s\n", prefix, timeStr, dryRunPrefix, r.Message, attrStr)

	if !h.opts.NoColor {
		line = colorize(r.Level, line)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, line)
	return err
}

// WithAttrs は属性を追加した新しいハンドラを返す。
func (h *HumanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &HumanHandler{
		w:      h.w,
		opts:   h.opts,
		attrs:  newAttrs,
		groups: h.groups,
		mu:     h.mu,
	}
}

// WithGroup はグループを追加した新しいハンドラを返す。
func (h *HumanHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &HumanHandler{
		w:      h.w,
		opts:   h.opts,
		attrs:  h.attrs,
		groups: newGroups,
		mu:     h.mu,
	}
}

func levelPrefix(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "x "
	case level >= slog.LevelWarn:
		return "! "
	default:
		return "  "
	}
}

func colorize(level slog.Level, line string) string {
	switch {
	case level >= slog.LevelError:
		return "\033[31m" + line + "\033[0m"
	case level >= slog.LevelWarn:
		return "\033[33m" + line + "\033[0m"
	case level >= slog.LevelInfo:
		return line
	default:
		// Debug: gray
		return "\033[90m" + line + "\033[0m"
	}
}
