package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
)

type FileWriter struct {
	file io.Writer
}

func NewFileWriter(name string, logDir string) io.Writer {
	log := Get(fmt.Sprintf("%s.ByteWriter", name))
	// Configure log rotation for this process
	logRoot := logDir
	if logRoot == "" {
		var err error
		logRoot, err = os.Getwd()
		if err != nil {
			log.Fatal("failed to get current working directory", zap.Error(err))
		}
	}
	logFile := filepath.Join(logRoot, fmt.Sprintf("%s.log", name))
	lumberjackLogger := &lumberjack.Logger{
		Filename: logFile,
	}

	return &FileWriter{
		file: lumberjackLogger,
	}
}

func (b *FileWriter) Write(p []byte) (n int, err error) {
	return b.file.Write(p)
}
