package system_error

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

const (
	ErrInvalidData = 1 // bad input/data; warn and continue
	ErrNoResponse  = 2 // a peer/request did not answer; warn and continue
	ErrFatal       = 3 // the daemon cannot continue
)

type SystemError struct {
	errBuff chan DaemonError
	log     *slog.Logger
}

type DaemonError struct {
	Message   string
	Trace     string
	ErrorCode int
	Err       error
}

func (e DaemonError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e DaemonError) Unwrap() error { return e.Err }

func InitSystemError(log *slog.Logger) *SystemError {
	return &SystemError{
		errBuff: make(chan DaemonError),
		log:     log,
	}
}

func (r *SystemError) NewError(ctx context.Context, message, trace string, errorCode int, err error) {
	daemonErr := DaemonError{
		Message:   message,
		Trace:     trace,
		ErrorCode: errorCode,
		Err:       err,
	}

	r.logError(ctx, daemonErr)

	select {
	case r.errBuff <- daemonErr:
	case <-ctx.Done():
	}
}

// logError logs with the source of NewError's caller, not this file.
func (r *SystemError) logError(ctx context.Context, e DaemonError) {
	level := slog.LevelWarn
	if e.ErrorCode != ErrInvalidData && e.ErrorCode != ErrNoResponse {
		level = slog.LevelError // ErrFatal, or an unknown code
	}

	if !r.log.Enabled(ctx, level) {
		return
	}

	// Skip runtime.Callers, logError and NewError.
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	rec := slog.NewRecord(time.Now(), level, e.Message, pcs[0])
	rec.AddAttrs(slog.Int("code", e.ErrorCode))
	if e.Trace != "" {
		rec.AddAttrs(slog.String("trace", e.Trace))
	}
	if e.Err != nil {
		rec.AddAttrs(slog.Any("err", e.Err))
	}
	_ = r.log.Handler().Handle(ctx, rec)
}

func (r *SystemError) HandleError(ctx context.Context, shutdown context.CancelCauseFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-r.errBuff:
			switch e.ErrorCode {
			case ErrInvalidData, ErrNoResponse:
				continue
			case ErrFatal:
				r.log.Error("fatal error, shutting down daemon", "error", e.Message)
				shutdown(e)
				return
			default:
				r.log.Warn("error with unknown code, continuing", "code", e.ErrorCode, "error", e.Message)
			}
		}
	}
}
