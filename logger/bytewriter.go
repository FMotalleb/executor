package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
)

type FileWriter struct {
	name          string
	output        io.Writer
	hasNamePrefix bool
}

func NewFileWriter(name string, logDir string) io.Writer {
	log := Get(name + ".ByteWriter")
	// Configure log rotation for this process
	logRoot := logDir
	if logRoot == "" {
		var err error
		logRoot, err = os.Getwd()
		if err != nil {
			log.Fatal("failed to get current working directory", zap.Error(err))
		}
	}
	logFile := filepath.Join(logRoot, name+".log")
	lumberjackLogger := &lumberjack.Logger{
		Filename: logFile,
	}

	return &FileWriter{
		name:          name,
		hasNamePrefix: false,
		output:        lumberjackLogger,
	}
}

func NewStdErrWriter(name string) io.Writer {
	return &FileWriter{
		name:          name,
		hasNamePrefix: true,
		output:        os.Stderr,
	}
}

func (b *FileWriter) Write(p []byte) (n int, err error) {
	buff := p
	if b.hasNamePrefix {
		buff = append([]byte(b.name+"|> "), buff...)
	}
	if n, err := b.output.Write(buff); err != nil {
		return n, err
	}
	return len(p), nil
}
