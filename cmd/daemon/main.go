package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/AndreeJait/go-utility/v2/gracefulw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/homelytics-agent/config"
)

func main() {
	configFlag := flag.String("config", "files/config/app.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logw.Init(&logw.LogConfig{
		Level:       cfg.Log.Level,
		Format:      cfg.Log.Format,
		WriteToFile: cfg.Log.WriteToFile,
		FilePath:    cfg.Log.FilePath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logw.Infof("Starting %s daemon", cfg.App.Name)

	ctx := context.Background()
	server, cleanup, err := wire(cfg)
	if err != nil {
		logw.Errorf("failed to wire dependencies: %v", err)
		os.Exit(1)
	}

	gracefulw.Register("ipc-server", func(ctx context.Context) error { return cleanup(ctx) })

	logw.Infof("Daemon IPC server listening on %s", cfg.IPC.SocketPath)
	gracefulw.Start(func() error { return server.Serve(ctx) }, cfg.Graceful.ShutdownTimeout)
}
