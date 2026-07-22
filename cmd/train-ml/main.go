package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	actions.ActiveConfig = config.Default()
	actions.SetDPIAware()

	// auto-detect data dir from standard paths
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "AppData", "Roaming", "go-mcp-computer-use", "datalog"),
	}

	var dataDir string
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "datalog.db")); err == nil {
			dataDir = c
			break
		}
	}
	if dataDir == "" {
		log.Fatal("datalog.db not found in any standard location")
	}

	fmt.Printf("Data dir: %s\n", dataDir)
	engine := actions.NewMLEngine(dataDir)

	fmt.Println("Training transformer from datalog...")
	if err := engine.Train(); err != nil {
		log.Fatalf("Train: %v", err)
	}

	fmt.Println("Done. Model ready:", engine.IsReady())
}
