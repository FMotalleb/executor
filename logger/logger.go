package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func Get(name string) *zap.Logger {
	return logger.Named(name)
}

func Initialize(isVerbose bool) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	lvl := zap.InfoLevel
	if isVerbose {
		lvl = zap.DebugLevel
	}
	// Create core for process logger
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		os.Stdout,
		lvl,
	)
	logger = zap.New(core)
}
