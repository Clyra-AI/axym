package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(executeContext(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
