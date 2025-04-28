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

var cfg executor.Config

const (
	defaultTimeoutH    = 24
	defaultBatchSize   = 1000
	defaultWorkerCount = 10
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "executor",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logger.Initialize()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
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
		"echo {{ .limit }} {{ .offset }}",
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
}
