package logger

import (
	"laguna/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultZapLevel = zapcore.InfoLevel
	defaultOut      = "stdout"
)

func New(c config.Logger) Logger {
	cfg := zap.NewDevelopmentConfig()

	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var level zapcore.Level
	switch c.Level {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = defaultZapLevel
	}

	cfg.DisableStacktrace = c.DisableStacktrace
	cfg.Level = zap.NewAtomicLevelAt(level)

	if c.Out == "" {
		cfg.OutputPaths = []string{defaultOut}
	} else {
		cfg.OutputPaths = []string{c.Out}
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("cannot build logger: " + err.Error())
	}

	return logger
}
