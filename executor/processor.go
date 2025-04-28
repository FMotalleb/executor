package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/FMotalleb/executor/logger"
	"github.com/FMotalleb/executor/template"
	"go.uber.org/zap"
)

type ExecRequest struct {
	RootCtx context.Context

	Command   string
	Offset    int
	BatchSize int

	Shell            string
	ShellArgs        []string
	WorkingDirectory string

	timeout time.Duration
	LogRoot string
}

func (e *ExecRequest) getVarMap() map[string]any {
	return map[string]any{
		"cmd":       e.Command,
		"offset":    e.Offset,
		"batchSize": e.BatchSize,
		"limit":     e.Offset + e.BatchSize,
	}
}

func processor(wg *sync.WaitGroup, requests <-chan *ExecRequest) {
	log := logger.Get("Processor")
	for r := range requests {
		rLog := log.With(zap.Any("request", r))
		cmd, err := template.EvaluateTemplate(r.Command, r.getVarMap())
		if err != nil {
			rLog.Error(
				"failed to evaluate template on given command",
				zap.Error(err),
			)
			continue
		}
		args := r.ShellArgs
		args = append(args, cmd)
		ctx, cancel := context.WithTimeout(r.RootCtx, r.timeout)
		name := fmt.Sprintf("exec-%d-%d", r.Offset, r.BatchSize)
		out := logger.NewFileWriter(name, r.LogRoot)
		err = spawnProcess(
			ctx,
			name,
			r.Shell,
			args,
			r.WorkingDirectory,
			out,
		)
		cancel()
		if err != nil {
			rLog.Error("failed to complete request")
		}
		wg.Done()
	}
}

func spawnProcess(
	ctx context.Context,
	name string,
	program string,
	args []string,
	wd string,
	out io.Writer,
) error {
	log := logger.Get("Spawner." + name)
	stdout, stderr, err := buildOutputPipes(out)
	if err != nil {
		return err
	}
	attr := &os.ProcAttr{
		Dir: wd,
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			stdout,
			stderr,
		},
	}
	proc, err := os.StartProcess(program, args, attr)
	if err != nil {
		return err
	}
	sigChan := make(chan int)
	go func() {
		stat, err := proc.Wait()
		if err != nil {
			log.Error("spawner failed to wait for process's exit", zap.Error(err))
			sigChan <- -1
			return
		}
		sigChan <- stat.ExitCode()
	}()

	select {
	case exitCode := <-sigChan:
		if exitCode != 0 {
			log.Error("process exited with status code!=0", zap.Int("code", exitCode))
		}
		return nil
	case <-ctx.Done():
		log.Warn("process timeout reached")
		return proc.Kill()
	}
}

func buildOutputPipes(out io.Writer) (*os.File, *os.File, error) {
	log := logger.Get("OutputPipes")
	oR, oW, oErr := os.Pipe()
	if oErr != nil {
		return nil, nil, oErr
	}
	eR, eW, eErr := os.Pipe()
	if eErr != nil {
		return nil, nil, eErr
	}
	go func() {
		_, err := io.Copy(out, oR)
		if err != nil {
			log.Error("failed to write stdout to file", zap.Error(err))
		}
	}()
	go func() {
		_, err := io.Copy(out, eR)
		if err != nil {
			log.Error("failed to write stderr to file", zap.Error(err))
		}
	}()
	return oW, eW, nil
}
