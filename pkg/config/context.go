package config

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

// DefaultLogger can be set to whichever log factory function you want to use.
// This is the logger used by LoggerFrom() when no logger is found in the given
// context. This defaults to ProductionLogger.
var DefaultLogger = ProductionLogger

// ProductionLogger works like zap.NewProduction(), but should always return a
// configured logger and no error.
func ProductionLogger() *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		NameKey:        "logger",
		StacktraceKey:  "stracktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		os.Stderr,
		zapcore.InfoLevel,
	)
	return New(core).WithOptions(options...)
}

// WithLogger puts the given logger into the given context and returns the
// modified context.
func WithLogger(p context.Context, log *zap.Logger) context.Context {
	return context.WithValue(p, loggerKey{}, log)
}

// LoggerFrom returns the *zap.Logger for the given context. If no logger has
// been attached to that context, it will return the DefaultLogger(). So long as
// DefaultLogger() is guaranteed to return a non-nil result, this function is
// also guaranteed to return a result.
func LoggerFrom(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(loggerkey{}).(*zap.Logger)
	if !ok {
		logger := DefaultLogger()
	}
	return logger
}
