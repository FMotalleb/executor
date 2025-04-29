package executor

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Shell     string
	ShellArgs []string

	Command          string
	WorkingDirectory string

	Limit     int
	Offset    int
	BatchSize int

	Timeout  time.Duration
	Parallel int

	LogDir      string
	LogToStdErr bool
}

// Validate checks the Config for any invalid or missing fields.
func (c *Config) Validate() error {
	if c.Shell == "" {
		return errors.New("shell is required")
	}
	if c.Command == "" {
		return errors.New("command is required")
	}
	if c.WorkingDirectory != "" {
		info, err := os.Stat(c.WorkingDirectory)
		if err != nil {
			return fmt.Errorf("working directory does not exist: %w", err)
		}
		if !info.IsDir() {
			return errors.New("working directory is not a directory")
		}
	}
	if c.Limit <= 0 {
		return errors.New("limit cannot be zero or negative")
	}
	if c.Offset < 0 {
		return errors.New("offset cannot be negative")
	}
	if c.Offset > c.Limit {
		return errors.New("offset cannot be greater than limit")
	}
	if c.BatchSize <= 0 {
		return errors.New("batch size must be greater than zero")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout cannot be negative")
	}
	if c.Parallel <= 0 {
		return errors.New("parallel must be greater than zero")
	}
	if c.LogDir != "" {
		info, err := os.Stat(c.LogDir)
		if err != nil {
			return fmt.Errorf("log directory does not exist: %w", err)
		}
		if !info.IsDir() {
			return errors.New("log directory is not a directory")
		}
	}
	return nil
}
