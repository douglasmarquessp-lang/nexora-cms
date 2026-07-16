package logger

import (
	"context"
	"log/slog"
	"os"
	"time"

	"nexora/internal/pkg/config"
)

type Logger struct {
	*slog.Logger
}

func New(cfg *config.Config) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					source.File = trimSourcePath(source.File)
				}
			}
			return a
		},
		AddSource: cfg.Debug,
	}

	switch cfg.LogFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{slog.New(handler)}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.With("error", err.Error())}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{l.With("component", component)}
}

func (l *Logger) WithRequestID(id string) *Logger {
	return &Logger{l.With("request_id", id)}
}

func (l *Logger) WithUser(id string) *Logger {
	return &Logger{l.With("user_id", id)}
}

func (l *Logger) WithSite(id string) *Logger {
	return &Logger{l.With("site_id", id)}
}

type contextKey string

const loggerKey contextKey = "logger"

func ToContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return nil
}

type LogFunc func(msg string, args ...any)

func (l *Logger) LogSlowQuery(duration time.Duration, query string, args []any) {
	if duration > 100*time.Millisecond {
		l.Warn("slow query detected",
			"duration_ms", duration.Milliseconds(),
			"query", query,
			"args", args,
		)
	}
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func trimSourcePath(path string) string {
	if len(path) <= 40 {
		return path
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			short := path[i+1:]
			if len(short) <= 40 {
				return "..." + short
			}
		}
	}
	return "..." + path[len(path)-37:]
}
