package executor

import (
	"context"
	"errors"
	"sync"

	"github.com/FMotalleb/executor/logger"
	"go.uber.org/zap"
)

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
	for i := begin; i <= end; i += stepSize {
		wg.Add(1)
		offset := i
		limit := stepSize
		if offset+limit > end {
			limit = ((offset + limit) - end)
		}
		reqChannel <- &ExecRequest{
			Command:   cfg.Command,
			Offset:    offset,
			BatchSize: limit,

			Shell:     cfg.Shell,
			ShellArgs: cfg.ShellArgs,

			WorkingDirectory: cfg.WorkingDirectory,
			LogRoot:          cfg.LogDir,

			RootCtx: ctx,
			timeout: cfg.Timeout,
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

func asChan(fn func()) <-chan any {
	ch := make(chan any)
	go func() {
		fn()
		ch <- nil
	}()
	return ch
}
