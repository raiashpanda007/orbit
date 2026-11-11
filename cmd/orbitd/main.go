// Command orbitd is the Orbit background daemon.
//
// TODO(ashwin): placeholder so the daemon binary builds and uses the shared
// logger. Replace with the real daemon lifecycle (HTTP server, store, node
// identity, ...).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/raiashpanda007/orbit/internal/logger"
)

func main() {
	os.Exit(run())
}

// run exists so deferred cleanup happens before os.Exit.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logFile, err := logger.DefaultFilePath("orbitd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "orbitd: resolve log file path: %v\n", err)
		return 1
	}

	// TODO(ashwin): daemon config (flags/env/file) is yours; hardcoded for now.
	log, err := logger.New(logger.Config{
		Level:    "info",
		Output:   logger.OutputBoth,
		FilePath: logFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "orbitd: init logger: %v\n", err)
		return 1
	}
	defer log.Close()

	log.Info("orbitd started", "pid", os.Getpid(), "log_file", logFile)
	<-ctx.Done()
	log.Info("orbitd shutting down", "reason", context.Cause(ctx))
	return 0
}
