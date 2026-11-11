package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	system_error "github.com/raiashpanda007/orbit/internal/error"
	"github.com/raiashpanda007/orbit/internal/logger"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logFile, err := logger.DefaultFilePath("orbitd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "orbitd: resolve log file path: %v\n", err)
		return 1
	}

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

	// A fatal daemon error cancels ctx (with the error as its cause), the same
	// way SIGINT/SIGTERM does, so there is one shutdown path.
	ctx, shutdown := context.WithCancelCause(ctx)

	// One error system for the whole daemon; pass it (with ctx) into the
	// master/worker daemons.
	sysErr := system_error.InitSystemError(log.Logger)

	var wg sync.WaitGroup
	wg.Go(func() { sysErr.HandleError(ctx, shutdown) })
	defer func() {
		shutdown(nil)
		wg.Wait()
	}()

	if len(os.Args) < 1 {
		log.Error("Not specified daemon role")
		return 0
	}

	role := os.Args[1]

	if role == "master" {
		log.Info("orbitd started as master")
	} else if role == "worker" {
		log.Info("orbitd started as worker")
	} else {
		log.Error("Invalid role to start daemon")
		return 0
	}

	log.Info("orbitd started", "pid", os.Getpid(), "log_file", logFile)
	<-ctx.Done()
	cause := context.Cause(ctx)
	log.Info("orbitd shutting down", "reason", cause)

	if _, ok := errors.AsType[system_error.DaemonError](cause); ok {
		return 1 // stopped by a fatal daemon error, not a signal
	}
	return 0
}
