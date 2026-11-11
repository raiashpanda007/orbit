// Package cli builds the orbit command-line interface.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/raiashpanda007/orbit/internal/logger"
)

// app holds what every command needs. Subcommand constructors take *app and
// use a.log.Logger and cmd.Context().
type app struct {
	logLevel  string
	logOutput string
	logFile   string

	log *logger.Logger
}

// Execute runs the orbit CLI with args and returns the process exit code.
// ctx is the root context; it is cancelled on SIGINT/SIGTERM.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	a := &app{}
	defer func() {
		if err := a.log.Close(); err != nil {
			fmt.Fprintf(stderr, "Error: close log file: %v\n", err)
		}
	}()

	root := newRootCmd(a)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	if err := root.ExecuteContext(ctx); err != nil {
		return 1
	}
	return 0
}

func newRootCmd(a *app) *cobra.Command {
	defaultLogFile, err := logger.DefaultFilePath("orbit")
	if err != nil {
		defaultLogFile = "" // --log-file must then be given for file/both output
	}

	cmd := &cobra.Command{
		Use:   "orbit",
		Short: "Manage the machines on your local network as one cloud",
		Long: `Orbit connects machines on the same local network and manages them as
one computing environment. A master node discovers and coordinates worker
nodes; the orbit CLI talks to the orbitd daemon running on this machine.

Orbit is in early development. No node commands exist yet.`,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		// No subcommands yet: print help. Having a RunE also makes cobra run
		// PersistentPreRunE, so the log flags are validated on a bare `orbit`.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			output, err := logger.ParseOutput(a.logOutput)
			if err != nil {
				return err
			}
			log, err := logger.New(logger.Config{
				Level:    a.logLevel,
				Output:   output,
				FilePath: a.logFile,
				Console:  cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			a.log = log
			a.log.Debug("logger ready", "log_level", a.logLevel, "log_output", output, "log_file", a.logFile)
			return nil
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&a.logLevel, "log-level", "warn", "log level: debug, info, warn, error")
	flags.StringVar(&a.logOutput, "log-output", string(logger.OutputConsole), "log output: console, file, both")
	flags.StringVar(&a.logFile, "log-file", defaultLogFile, "log file path, used when --log-output is file or both")

	return cmd
}
