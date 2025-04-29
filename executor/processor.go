package executor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
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

	timeout  time.Duration
	LogRoot  string
	LogToErr bool
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
		rLog := log.With(
			zap.Any("request", r),
		)

		rLog.Debug("received request for processing")

		cmd, err := template.EvaluateTemplate(r.Command, r.getVarMap())
		if err != nil {
			rLog.Error(
				"failed to evaluate command template",
				zap.Error(err),
				zap.String("raw_command", r.Command),
			)
			continue
		}

		rLog.Debug("successfully evaluated command template", zap.String("evaluated_command", cmd))

		args := append(r.ShellArgs, cmd)
		ctx, cancel := context.WithTimeout(r.RootCtx, r.timeout)

		name := fmt.Sprintf("exec-%d-%d", r.Offset, r.BatchSize)
		var out io.Writer
		if r.LogToErr {
			out = logger.NewStdErrWriter(name)
		} else {
			out = logger.NewFileWriter(name, r.LogRoot)
		}

		rLog.Debug(
			"spawning process",
			zap.String("process_name", name),
			zap.String("shell", r.Shell),
			zap.Strings("args", args),
			zap.String("working_directory", r.WorkingDirectory),
		)

		err = spawnProcess(
			ctx,
			name,
			r.Shell,
			args,
			r.WorkingDirectory,
			out,
		)

		if err != nil {
			rLog.Error(
				"process execution failed",
				zap.Error(err),
				zap.String("process_name", name),
			)
		} else {
			rLog.Info(
				"process execution completed successfully",
				zap.String("process_name", name),
			)
		}
		cancel()
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
	log := logger.Get("Spawner."+name).With(
		zap.String("program", program),
		zap.Strings("args", args),
		zap.String("working_directory", wd),
	)

	log.Debug("starting process setup")

	log.Debug("attempting to start process")
	proc := exec.CommandContext(ctx, program, args...)

	err := connectPipes(proc, out)
	if err != nil {
		log.Error("failed to build output pipes", zap.Error(err))
		return err
	}

	err = proc.Start()
	if err != nil {
		log.Error("failed to start process", zap.Error(err))
		return err
	}

	log.Info("process started successfully", zap.Int("pid", proc.Process.Pid))

	sigChan := make(chan int)
	go func() {
		stat, err := proc.Process.Wait()
		if err != nil {
			log.Error("failed to wait for process exit", zap.Error(err))
			sigChan <- -1
			return
		}
		exitCode := stat.ExitCode()
		log.Debug("process exited", zap.Int("exit_code", exitCode))
		sigChan <- exitCode
	}()

	select {
	case exitCode := <-sigChan:
		if exitCode != 0 {
			log.Error("process exited with non-zero status", zap.Int("exit_code", exitCode))
		} else {
			log.Info("process exited cleanly", zap.Int("exit_code", exitCode))
		}
		return nil
	}
}

func connectPipes(proc *exec.Cmd, out io.Writer) error {
	log := logger.Get("OutputPipes")
	oR, oErr := proc.StdoutPipe()
	if oErr != nil {
		return oErr
	}
	eR, eErr := proc.StderrPipe()
	if eErr != nil {
		return eErr
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
	return nil
}
