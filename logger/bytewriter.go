package logger

import (
	"bytes"
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
	lines := bytes.Split(p, []byte("\n"))
	var buff []byte
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if b.hasNamePrefix {
			buff = append(buff, []byte(b.name+"|> ")...)
		}
		buff = append(buff, line...)
		buff = append(buff, '\n')
	}
	if n, err := b.output.Write(buff); err != nil {
		return n, err
	}
	return len(p), nil
}
