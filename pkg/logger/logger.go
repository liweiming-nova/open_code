package logger

import (
	"context"

	"github.com/liweiming-nova/open_code/pkg/context/trace"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func init() {
	zapCfg := zap.NewProductionConfig()
	zapCfg.Encoding = "json"

	zapCfg.EncoderConfig.TimeKey = "time"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapCfg.EncoderConfig.MessageKey = "title"

	logger, _ = zapCfg.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	)
}

func withContext(ctx context.Context) *zap.Logger {
	traceID := trace.GetTraceID(ctx)
	return logger.With(zap.String("trace_id", traceID))
}

func log(ctx context.Context, level zapcore.Level, title string, fields ...zap.Field) {
	withContext(ctx).Log(level, title, fields...)
}

func Sync() error {
	if logger == nil {
		return nil
	}
	return logger.Sync()
}

func Info(ctx context.Context, title string, fields ...zap.Field) {
	log(ctx, zap.InfoLevel, title, fields...)
}

func Error(ctx context.Context, title string, fields ...zap.Field) {
	log(ctx, zap.ErrorLevel, title, fields...)
}

func Warn(ctx context.Context, title string, fields ...zap.Field) {
	log(ctx, zap.WarnLevel, title, fields...)
}

func Debug(ctx context.Context, title string, fields ...zap.Field) {
	log(ctx, zap.DebugLevel, title, fields...)
}

func Fatal(ctx context.Context, title string, fields ...zap.Field) {
	log(ctx, zap.FatalLevel, title, fields...)
}
