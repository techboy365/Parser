package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

func New(level, format string) *slog.Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type JobLogger struct {
	base   *slog.Logger
	jobID  string
	dorkID string
}

func NewJobLogger(base *slog.Logger, jobID, dorkID string) *JobLogger {
	return &JobLogger{base: base, jobID: jobID, dorkID: dorkID}
}

func (l *JobLogger) attrs() []any {
	return []any{"job_id", l.jobID, "dork_id", l.dorkID}
}

func (l *JobLogger) Info(msg string, args ...any) {
	l.base.Info(msg, append(l.attrs(), args...)...)
}

func (l *JobLogger) Warn(msg string, args ...any) {
	l.base.Warn(msg, append(l.attrs(), args...)...)
}

func (l *JobLogger) Error(msg string, args ...any) {
	l.base.Error(msg, append(l.attrs(), args...)...)
}

func (l *JobLogger) Debug(msg string, args ...any) {
	l.base.Debug(msg, append(l.attrs(), args...)...)
}

func TeeWriter(path string) (io.Writer, func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return io.MultiWriter(os.Stdout, f), f.Close, nil
}

func Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
