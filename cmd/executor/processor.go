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

// ExecRequest represents a request to execute a command with specific parameters.
// It contains configuration options for the execution environment, command details,
// and logging preferences.
//
// Fields:
// - rootCtx: The root context for the execution, used for cancellation and deadlines.
// - Command: The command to be executed.
// - Offset: The starting offset for processing, if applicable.
// - BatchSize: The size of the batch to process, if applicable.
// - Shell: The shell to use for executing the command.
// - ShellArgs: Additional arguments to pass to the shell.
// - WorkingDirectory: The directory in which the command should be executed.
// - Timeout: The maximum duration allowed for the command execution.
// - logRoot: The root directory for storing logs.
// - logToErr: A flag indicating whether logs should also be written to stderr.
type ExecRequest struct {
	rootCtx context.Context

	Command   string
	StdIn     string
	Offset    int
	BatchSize int

	Shell            string
	ShellArgs        []string
	WorkingDirectory string

	Timeout  time.Duration
	logRoot  string
	logToErr bool
}

// getVarMap to be used in template engine.
func (e *ExecRequest) getVarMap() map[string]any {
	return map[string]any{
		"offset":    e.Offset,
		"batchSize": e.BatchSize,
		"limit":     e.Offset + e.BatchSize,
	}
}

// processor is a function that processes execution requests.
// It listens to a channel of ExecRequest objects, evaluates command templates,
// and spawns processes to execute the commands. The function logs the progress
// and results of each request, including errors and successful completions.
//
// Parameters:
//   - wg: A WaitGroup used to synchronize the completion of all processing tasks.
//   - requests: A receive-only channel of pointers to ExecRequest objects, which
//     contain the details of the commands to be executed.
//
// The function performs the following steps for each request:
//  1. Logs the receipt of the request.
//  2. Evaluates the command template using the request's variable map.
//  3. Prepares the command arguments and execution context.
//  4. Spawns a process to execute the command, directing output to either
//     a file or standard error based on the request's configuration.
//  5. Logs the outcome of the process execution (success or failure).
//
// The function ensures that the WaitGroup counter is decremented for each
// processed request, signaling its completion.
func processor(wg *sync.WaitGroup, requests <-chan *ExecRequest) {
	log := logger.Get("Processor")
	for r := range requests {
		process(log, r, wg)
	}
}

func process(log *zap.Logger, r *ExecRequest, wg *sync.WaitGroup) {
	defer wg.Done()
	rLog := log.With(
		zap.Any("request", r),
	)

	rLog.Debug("received request for processing")

	name, args, stdin, out, err := prepareArgs(rLog, r)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.rootCtx, r.Timeout)
	defer cancel()
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
		stdin,
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
}

func prepareArgs(rLog *zap.Logger, r *ExecRequest) (string, []string, string, io.Writer, error) {
	cmd, err := template.EvaluateTemplate(r.Command, r.getVarMap())
	if err != nil {
		rLog.Error(
			"failed to evaluate command template",
			zap.Error(err),
			zap.String("raw_command", r.Command),
		)
		return "", nil, "", nil, err
	}
	stdinVal, err := template.EvaluateTemplate(r.StdIn, r.getVarMap())
	if err != nil {
		rLog.Error(
			"failed to evaluate command template",
			zap.Error(err),
			zap.String("raw_command", r.Command),
		)
		return "", nil, "", nil, err
	}

	rLog.Debug("successfully evaluated command template", zap.String("evaluated_command", cmd))
	args := r.ShellArgs
	args = append(args, cmd)

	name := fmt.Sprintf("exec-%d-%d", r.Offset, r.BatchSize)
	var out io.Writer
	if r.logToErr {
		out = logger.NewStdErrWriter(name)
	} else {
		out = logger.NewFileWriter(name, r.logRoot)
	}
	return name, args, stdinVal, out, nil
}

func spawnProcess(
	ctx context.Context,
	name string,
	program string,
	args []string,
	wd string,
	stdin string,
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

	err := connectPipes(proc, out, stdin)
	if err != nil {
		log.Error("failed to build output pipes", zap.Error(err))
		return err
	}

	sigChan := make(chan int)
	go spawnSubprocess(proc, log, sigChan)

	if ec := <-sigChan; ec != 0 {
		log.Error("process exited with non-zero status", zap.Int("exit_code", ec))
		return fmt.Errorf("process exited with non-zero status: %d", ec)
	}
	log.Info("process exited cleanly", zap.Int("exit_code", 0))
	return nil
}

func spawnSubprocess(proc *exec.Cmd, log *zap.Logger, sigChan chan int) {
	err := proc.Start()
	if err != nil {
		log.Error("failed to start process", zap.Error(err))
		sigChan <- 1
		return
	}

	log.Info("process started successfully", zap.Int("pid", proc.Process.Pid))

	stat, err := proc.Process.Wait()
	if err != nil {
		log.Error("failed to wait for process exit", zap.Error(err))
		sigChan <- -1
		return
	}
	exitCode := stat.ExitCode()
	log.Debug("process exited", zap.Int("exit_code", exitCode))
	sigChan <- exitCode
}

func connectPipes(proc *exec.Cmd, out io.Writer, stdin string) error {
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
	iW, iErr := proc.StdinPipe()

	if iErr != nil {
		return iErr
	}
	go func() {
		data := []byte(stdin)
		mustWrite := len(data)
		totalWrites := 0
		for totalWrites < mustWrite {
			n, err := iW.Write(data[totalWrites:mustWrite])
			if err != nil {
				log.Error("failed to write stdin to process", zap.Error(err))
			}
			totalWrites += n
		}
		if err := iW.Close(); err != nil {
			log.Error("failed to close stdin", zap.Error(err))
		}
	}()

	return nil
}
