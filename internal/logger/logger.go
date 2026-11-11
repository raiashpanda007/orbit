// Package logger builds the structured slog logger shared by the orbit CLI
// and the orbitd daemon.
//
// A logger writes to the console, to a log file, or to both. Every line
// carries a timestamp, a level and the source file:line of the call.
package logger

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Output selects where log lines are written.
type Output string

const (
	OutputConsole Output = "console"
	OutputFile    Output = "file"
	OutputBoth    Output = "both"
)

const timeFormat = "2006-01-02T15:04:05.000Z07:00"

// Config describes how to build a Logger.
type Config struct {
	// Level is the minimum level logged: debug, info, warn or error
	// (case-insensitive). Empty means info.
	Level string

	// Output is console, file or both. Empty means console.
	Output Output

	// FilePath is the log file. Required when Output is file or both.
	// Missing parent directories are created.
	FilePath string

	// Console is where console output goes. Nil means os.Stderr, which keeps
	// stdout free for command output.
	Console io.Writer
}

// Logger is a *slog.Logger plus the log file it may own.
//
// Pass the embedded *slog.Logger (l.Logger) to other packages so they depend
// only on log/slog. Do not add logging methods to this type: a wrapper method
// would make every "source=" point at this file instead of the caller.
type Logger struct {
	*slog.Logger

	file      *os.File
	closeOnce sync.Once
	closeErr  error
}

// New builds a Logger from cfg. Call Close when done, typically deferred in
// main, so the log file is flushed and closed on shutdown.
func New(cfg Config) (*Logger, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	output := cfg.Output
	if output == "" {
		output = OutputConsole
	}
	if output, err = ParseOutput(string(output)); err != nil {
		return nil, err
	}

	console := cfg.Console
	if console == nil {
		console = os.Stderr
	}

	opts := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	l := &Logger{}
	var handler slog.Handler

	switch output {
	case OutputConsole:
		handler = slog.NewTextHandler(console, opts)
	case OutputFile, OutputBoth:
		f, err := openFile(cfg.FilePath)
		if err != nil {
			return nil, err
		}
		l.file = f
		handler = slog.NewTextHandler(f, opts)
		if output == OutputBoth {
			handler = slog.NewMultiHandler(slog.NewTextHandler(console, opts), handler)
		}
	}

	l.Logger = slog.New(handler)
	return l, nil
}

// Close flushes and closes the log file, if there is one. It is safe to call
// more than once and on a nil Logger.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		l.closeErr = errors.Join(l.file.Sync(), l.file.Close())
	})
	return l.closeErr
}

// ParseLevel converts debug, info, warn or error (case-insensitive) to a
// slog.Level. Empty means info.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", s)
	}
}

// ParseOutput converts console, file or both (case-insensitive) to an Output.
func ParseOutput(s string) (Output, error) {
	switch o := Output(strings.ToLower(strings.TrimSpace(s))); o {
	case OutputConsole, OutputFile, OutputBoth:
		return o, nil
	default:
		return "", fmt.Errorf("invalid log output %q: must be one of console, file, both", s)
	}
}

func openFile(path string) (*os.File, error) {
	if path == "" {
		return nil, errors.New("log file path is required when log output is file or both")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return f, nil
}

// replaceAttr formats the built-in time with milliseconds and shortens the
// source to "dir/file.go:line".
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		if t := a.Value.Time(); !t.IsZero() {
			return slog.String(slog.TimeKey, t.Format(timeFormat))
		}
	case slog.SourceKey:
		if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
			short := filepath.Join(filepath.Base(filepath.Dir(src.File)), filepath.Base(src.File))
			return slog.String(slog.SourceKey, short+":"+strconv.Itoa(src.Line))
		}
	}
	return a
}
