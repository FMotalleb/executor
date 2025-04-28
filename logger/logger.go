package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func Get(name string) *zap.Logger {
	return logger.Named(name)
}

func Initialize() error {
	var err error
	logger, err = zap.NewProduction(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	return nil
}

func GenerateLogger(name string) (*zap.Logger, error) {

	// Configure log rotation for this process
	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal("failed to get current working directory", zap.Error(err))
	}
	logFile := filepath.Join(wd, fmt.Sprintf("%s.log", name))
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    128,
		MaxBackups: 4,
		Compress:   true,
	}

	// Create Zap encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create core for process logger
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(lumberjackLogger),
		zap.InfoLevel,
	)

	// Create process logger
	processLogger := zap.New(core).Named(name)
	return processLogger, nil
}
