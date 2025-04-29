/*
Copyright © 2025 Motalleb Fallahnezhad

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/FMotalleb/executor/executor"
	"github.com/FMotalleb/executor/logger"
	"github.com/spf13/cobra"
)

var (
	cfg       executor.Config
	isVerbose bool
)

const (
	defaultTimeoutH    = 24
	defaultBatchSize   = 1000
	defaultWorkerCount = 10
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "executor",
	Short: "A CLI tool to orchestrate parallel batch processing",
	Long: `Executor is a command-line application designed to orchestrate 
and execute parallel processes with configurable batch size, offset, 
limit, and custom commands. It provides flexibility for managing 
multi-process workflows efficiently.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		logger.Initialize(isVerbose)
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx := executor.NewSystemContext()
		return executor.StartExecution(ctx, cfg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get current working directory: %w", err))
	}

	rootCmd.Flags().StringVar(
		&cfg.Shell,
		"shell",
		"/bin/sh",
		"Shell to use for executing commands",
	)

	rootCmd.Flags().StringSliceVar(
		&cfg.ShellArgs,
		"shell-args",
		[]string{"-c"},
		"Arguments to pass to the shell",
	)

	rootCmd.Flags().StringVarP(
		&cfg.Command,
		"command",
		"c",
		"echo {{ .offset | sum .batchSize  }}={{ .limit }} ",
		"Command to execute (evaluated as Go template with variables: cmd, offset, batchSize, limit)",
	)

	rootCmd.Flags().StringVarP(
		&cfg.WorkingDirectory,
		"working-directory",
		"w",
		wd,
		"Working directory for the command execution",
	)

	rootCmd.Flags().IntVarP(
		&cfg.Offset,
		"offset",
		"o",
		0,
		"Starting offset for processing",
	)

	rootCmd.Flags().IntVar(
		&cfg.BatchSize,
		"batch-size",
		defaultBatchSize,
		"Batch size for processing",
	)

	rootCmd.Flags().IntVarP(
		&cfg.Limit,
		"limit",
		"l",
		0,
		"Total limit of items to process",
	)

	rootCmd.Flags().DurationVar(
		&cfg.Timeout,
		"timeout",
		time.Hour*defaultTimeoutH,
		"Timeout for each command execution",
	)

	rootCmd.Flags().IntVarP(
		&cfg.Parallel,
		"processors",
		"p",
		defaultWorkerCount,
		"Number of parallel executions",
	)

	rootCmd.Flags().StringVar(&cfg.LogDir, "log-dir", wd, "Directory to store logs")
	rootCmd.Flags().BoolVar(&cfg.LogToStdErr, "log-stderr", false, "Log directly to stderr instead of file")

	rootCmd.
		PersistentFlags().
		BoolVarP(&isVerbose, "verbose", "v", false, "Changes logger to verbose")
}
