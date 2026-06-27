package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/go-mcp-computer-use/internal/actions"
	"github.com/user/go-mcp-computer-use/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: nil})))

	if err := actions.CheckScreenshotPermission(); err != nil {
		slog.Warn("screenshot may not work", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		os.Exit(0)
	}()

	slog.Info("starting on stdio")
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
