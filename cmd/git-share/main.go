package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

var Log *core.MultiLogger

func main() {
	consoleLog := core.NewHybridLogger("fluxor", false)
	fileLog, _ := core.NewFileAppendLogger(core.FileLoggerConfig{Service: "git-share", FilePath: "git0share.log"})
	defer fileLog.Close()
	Log = core.NewMultiLogger(consoleLog, fileLog)

	runtime.GOMAXPROCS(runtime.NumCPU())
	Log.Info(fmt.Sprintf("Runtime: Go %s, GOMAXPROCS=%d, NumCPU=%d", runtime.Version(), runtime.GOMAXPROCS(0), runtime.NumCPU()))

	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		Log.Error(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	app, err := entrypoint.NewMainVerticleWithOptions(configPath, entrypoint.WithOptions(entrypoint.MainVerticleOptions{}))
	if err != nil {
		Log.Error(fmt.Sprintf("Failed to create runtime: %v", err))
		os.Exit(1)
	}

	vert := NewGitShareVerticle(cfg)
	if _, err := app.DeployVerticle(vert); err != nil {
		Log.Error(fmt.Sprintf("Deploy error: %v", err))
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	errChan := make(chan error, 1)
	go func() {
		if err := app.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		Log.Error(fmt.Sprintf("Application error: %v", err))
		os.Exit(1)
	case sig := <-sigChan:
		Log.Info(fmt.Sprintf("Received %v, shutting down...", sig))
	}

	if err := app.Stop(); err != nil {
		Log.Error(fmt.Sprintf("Shutdown error: %v", err))
		os.Exit(1)
	}
	Log.Info("Server shutdown complete")
}
