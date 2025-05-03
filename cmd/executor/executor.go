package executor

import (
	"context"
	"errors"
	"sync"

	"github.com/FMotalleb/executor/logger"
	"go.uber.org/zap"
)

// StartExecution initializes and manages the execution of tasks based on the provided configuration.
// It validates the configuration, spawns worker goroutines, and processes tasks in batches.
//
// Parameters:
//   - ctx: A context.Context object used to manage the lifecycle of the execution process.
//   - cfg: A Config object containing the execution parameters such as parallelism, batch size, command, and more.
//
// Returns:
//   - error: Returns an error if the configuration is invalid or if the execution is prematurely terminated.
//
// Behavior:
//   - Validates the provided configuration. If invalid, logs the error and terminates.
//   - Creates a channel for execution requests and spawns worker goroutines based on the configured parallelism.
//   - Processes tasks in batches, sending execution requests to the worker goroutines.
//   - Monitors the context for cancellation and ensures proper cleanup of resources.
//   - Waits for all worker goroutines to complete before returning.
//
// Notes:
//   - The function ensures that the request channel is closed properly after use.
//   - If the context is canceled before completion, an error is returned, and the process is terminated.
func StartExecution(ctx context.Context, cfg Config) error {
	log := logger.Get("ExecutionController")
	if err := cfg.Validate(); err != nil {
		log.Fatal(
			"configuration is not valid",
			zap.Any("cfg", cfg),
			zap.Error(err),
		)
	}
	reqChannel := make(chan *ExecRequest)
	wg := new(sync.WaitGroup)
	defer close(reqChannel)
	for i := 0; i < cfg.Parallel; i++ {
		go processor(wg, reqChannel)
	}

	begin := cfg.Offset
	stepSize := cfg.BatchSize
	end := cfg.Limit
	for i := begin; i < end; i += stepSize {
		wg.Add(1)
		offset := i
		limit := stepSize
		if offset+limit > end {
			limit = end - offset
		}
		reqChannel <- &ExecRequest{
			Command:   cfg.Command,
			StdIn:     cfg.StdIn,
			Offset:    offset,
			BatchSize: limit,

			Shell:     cfg.Shell,
			ShellArgs: cfg.ShellArgs,

			WorkingDirectory: cfg.WorkingDirectory,
			logRoot:          cfg.LogDir,

			rootCtx: ctx,
			Timeout: cfg.Timeout,

			logToErr: cfg.LogToStdErr,
		}
	}

	select {
	case <-ctx.Done():
		log.Error("premature execution killed by a dead context")
		return errors.New("premature execution killed by a dead context")
	case <-asChan(wg.Wait):
		log.Info("process finished")
		return nil
	}
}

// asChan is here to convert a function into channel signal (like wg.Wait()) in order to be able to use select on it.
func asChan(fn func()) <-chan any {
	ch := make(chan any)
	go func() {
		fn()
		ch <- nil
	}()
	return ch
}
