package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
)

func main() {
	train := flag.Bool("train", false, "train transformer from datalog then evaluate")
	dataDir := flag.String("data", "", "datalog directory (default: %APPDATA%/go-mcp-computer-use/datalog)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	actions.ActiveConfig = config.Default()
	actions.SetDPIAware()

	dir := *dataDir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "AppData", "Roaming", "go-mcp-computer-use", "datalog")
	}
	dbPath := filepath.Join(dir, "datalog.db")
	if _, err := os.Stat(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "datalog.db not found at %s\n", dbPath)
		os.Exit(1)
	}
	fmt.Println("Data dir:", dir)

	loader := dataloader.NewSQLiteLoader(dbPath)
	defer loader.Close()
	samples, err := loader.LoadAll(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "load samples: %v\n", err)
		os.Exit(1)
	}

	tools := map[string]int{}
	withCoords := 0
	for _, s := range samples {
		tools[s.Action]++
		if s.CoordX != 0 || s.CoordY != 0 {
			withCoords++
		}
	}
	type kv struct {
		k string
		v int
	}
	var list []kv
	maj, majN := "", 0
	for k, v := range tools {
		list = append(list, kv{k, v})
		if v > majN {
			maj, majN = k, v
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })

	fmt.Println("=== DATASET ===")
	fmt.Printf("samples=%d with_coords=%d (%.1f%%)\n", len(samples), withCoords, pct(withCoords, len(samples)))
	fmt.Println("tool_dist:")
	for _, e := range list {
		fmt.Printf("  %s: %d (%.1f%%)\n", e.k, e.v, pct(e.v, len(samples)))
	}
	fmt.Printf("majority_baseline_tool_acc=%.2f%% (always '%s')\n", pct(majN, len(samples)), maj)

	// sample decode check
	if len(samples) > 0 {
		s := samples[0]
		fmt.Printf("sample0 action=%s coord=(%d,%d) args=%.120s\n", s.Action, s.CoordX, s.CoordY, s.ArgsJSON)
	}

	engine := actions.NewMLEngine(dir)
	if *train {
		fmt.Println("=== TRAIN ===")
		if err := engine.Train(); err != nil {
			fmt.Fprintf(os.Stderr, "train failed: %v\n", err)
			// still print status
		}
	} else {
		if err := engine.LoadModel(); err != nil {
			fmt.Printf("LoadModel: %v\n", err)
		}
	}

	st := engine.Status()
	fmt.Println("=== ML STATUS ===")
	b, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(b))

	// smoke predict on real OCR from datalog (not synthetic OOV text)
	var smokeOCR string
	for _, s := range samples {
		if s.Action == "click" && len(s.Context) > 20 {
			smokeOCR = s.Context
			break
		}
	}
	if smokeOCR == "" {
		smokeOCR = "Steam settings button Open file"
	}
	fmt.Println("IsReady:", engine.IsReady())
	fmt.Println("smoke_ocr:", smokeOCR)
	preds := engine.Predict(smokeOCR, 5, nil)
	fmt.Println("=== SMOKE PREDICT (real OCR) ===")
	pb, _ := json.MarshalIndent(preds, "", "  ")
	fmt.Println(string(pb))
	st2 := engine.Status()
	fmt.Println("status_after_predict last_error=", st2.LastError, "source=", st2.Source, "ready=", st2.Ready)

	// also synthetic OOV
	preds2 := engine.Predict("Steam settings button Open file", 5, nil)
	fmt.Println("=== SMOKE PREDICT (synthetic OOV) ===")
	pb2, _ := json.MarshalIndent(preds2, "", "  ")
	fmt.Println(string(pb2))

	if st.LastEvalAccuracy > 0 && st.MajorityBaseline > 0 {
		fmt.Printf("\nmodel_acc=%.2f%% majority=%.2f%% delta=%+.2fpp\n",
			st.LastEvalAccuracy*100, st.MajorityBaseline*100,
			(st.LastEvalAccuracy-st.MajorityBaseline)*100)
		if st.LastEvalAccuracy < st.MajorityBaseline {
			fmt.Println("WARNING: model below majority baseline — prefer statistical path / more data")
			os.Exit(2)
		}
	}
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}
