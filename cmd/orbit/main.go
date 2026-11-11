// Command orbit is the Orbit command-line interface.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/raiashpanda007/orbit/internal/cli"
)

func main() {
	os.Exit(run())
}

// run exists so deferred cleanup happens before os.Exit.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
