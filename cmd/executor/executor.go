package executor

import (
	"context"
	"errors"
	"sync"

	"github.com/FMotalleb/executor/logger"
	"go.uber.org/zap"
)

// StartExecution handles the initialization and management of task execution based on the provided configuration.
// It performs validation, creates worker goroutines, and processes tasks in batches until completion or cancellation.
//
// Parameters:
// - ctx: A context to control the lifecycle of the execution, including cancellation or timeout.
// - cfg: A Config object specifying execution parameters such as parallelism, batch size, retry limits, and more.
//
// Returns:
// - error: If there is a configuration validation failure or premature termination due to context cancellation.
//
// Behavior:
// - Validates the provided Config object to ensure correctness before execution starts.
// - Sets up a channel for execution requests and spawns a number of worker goroutines based on the configured parallelism.
// - Divides tasks into batches, creating and sending ExecRequest objects through the channel.
// - Continuously monitors the provided context for cancellation and performs cleanup if triggered.
// - Waits for all worker goroutines to finish execution before returning.
// - Ensures graceful shutdown by properly closing the request channel and synchronizing goroutines.
//
// Notes:
// - If the context is canceled before completion, the function terminates and returns an appropriate error.
// - Logging is used to record the process lifecycle, including errors and successful completion.
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

			Retry: cfg.Retry,

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
