package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Logger struct {
	log *slog.Logger
}

func NewLogger() *Logger {
	handler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "15:04:05",
		NoColor:    false,
	})

	l := slog.New(handler)

	return &Logger{
		log: l,
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		log: l.log.With(args...),
	}
}

func InitGlobalLogger() {
	logger := NewLogger()
	slog.SetDefault(logger.log)
	slog.Info("Logger initialized successfully")
}
