package actions

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ElementPrior struct {
	Class       string  `json:"class"`
	WindowTitle string  `json:"window_title"`
	Frequency   float64 `json:"frequency"`
	SampleCount int     `json:"sample_count"`
	AvgX        float64 `json:"avg_x"`
	AvgY        float64 `json:"avg_y"`
	AvgW        float64 `json:"avg_w"`
	AvgH        float64 `json:"avg_h"`
	StdX        float64 `json:"std_x"`
	StdY        float64 `json:"std_y"`
}

type priorCache struct {
	mu     sync.RWMutex
	priors []ElementPrior
	loaded bool
}

var elementPriors priorCache

func normalizeWindowTitle(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" {
		return "__empty__"
	}
	parts := strings.Fields(t)
	if len(parts) > 0 {
		first := parts[0]
		known := []string{"chrome", "firefox", "edge", "explorer", "code",
			"settings", "terminal", "notepad", "calculator", "outlook",
			"slack", "discord", "spotify", "word", "excel", "powerpoint",
			"control", "task", "file", "program"}
		for _, k := range known {
			if first == k {
				return first
			}
		}
	}
	if len(t) > 40 {
		return t[:40]
	}
	return t
}

func createPriorsTable(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS element_priors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			class TEXT NOT NULL,
			window_title TEXT NOT NULL,
			sample_count INTEGER NOT NULL DEFAULT 0,
			frequency REAL NOT NULL DEFAULT 0,
			sum_x REAL NOT NULL DEFAULT 0,
			sum_y REAL NOT NULL DEFAULT 0,
			sum_w REAL NOT NULL DEFAULT 0,
			sum_h REAL NOT NULL DEFAULT 0,
			sum_x2 REAL NOT NULL DEFAULT 0,
			sum_y2 REAL NOT NULL DEFAULT 0,
			total_frames INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			UNIQUE(class, window_title)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_priors_window ON element_priors(window_title)`,
		`CREATE INDEX IF NOT EXISTS idx_priors_class ON element_priors(class)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func InitPriors() error {
	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return err
		}
	}
	return createPriorsTable(trainDB)
}

func loadPriorsFromDB() {
	elementPriors.mu.Lock()
	defer elementPriors.mu.Unlock()

	if trainDB == nil {
		return
	}

	rows, err := trainDB.Query(`SELECT class, window_title, sample_count, frequency,
		CASE WHEN sample_count > 0 THEN sum_x / CAST(sample_count AS REAL) ELSE 0 END,
		CASE WHEN sample_count > 0 THEN sum_y / CAST(sample_count AS REAL) ELSE 0 END,
		CASE WHEN sample_count > 0 THEN sum_w / CAST(sample_count AS REAL) ELSE 0 END,
		CASE WHEN sample_count > 0 THEN sum_h / CAST(sample_count AS REAL) ELSE 0 END,
		CASE WHEN sample_count > 1 THEN
			SQRT((sum_x2 - (sum_x * sum_x / CAST(sample_count AS REAL))) / CAST((sample_count - 1) AS REAL))
		ELSE 0 END,
		CASE WHEN sample_count > 1 THEN
			SQRT((sum_y2 - (sum_y * sum_y / CAST(sample_count AS REAL))) / CAST((sample_count - 1) AS REAL))
		ELSE 0 END
		FROM element_priors WHERE sample_count > 0`)
	if err != nil {
		return
	}
	defer rows.Close()

	var priors []ElementPrior
	for rows.Next() {
		var p ElementPrior
		if err := rows.Scan(&p.Class, &p.WindowTitle, &p.SampleCount, &p.Frequency,
			&p.AvgX, &p.AvgY, &p.AvgW, &p.AvgH, &p.StdX, &p.StdY); err != nil {
			continue
		}
		priors = append(priors, p)
	}
	elementPriors.priors = priors
	elementPriors.loaded = true
}

func UpdatePriorsFromDetections(windowTitle string, elements []DetectedElement) {
	elementPriors.mu.Lock()
	defer elementPriors.mu.Unlock()

	if trainDB == nil {
		if err := InitPriors(); err != nil {
			return
		}
	}

	normalized := normalizeWindowTitle(windowTitle)
	tx, err := trainDB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	seen := make(map[string]bool)
	for _, el := range elements {
		class := el.Class
		key := class + "|" + normalized
		if seen[key] {
			continue
		}
		seen[key] = true

		xNorm := float64(el.X+el.W/2) / 100.0
		yNorm := float64(el.Y+el.H/2) / 100.0
		wNorm := float64(el.W) / 100.0
		hNorm := float64(el.H) / 100.0

		_, err := tx.Exec(`INSERT INTO element_priors(class, window_title, sample_count, frequency,
			sum_x, sum_y, sum_w, sum_h, sum_x2, sum_y2, total_frames, updated_at)
			VALUES(?, ?, 1, 1.0, ?, ?, ?, ?, ?, ?, 1, datetime('now'))
			ON CONFLICT(class, window_title) DO UPDATE SET
				sample_count = sample_count + 1,
				sum_x = sum_x + ?,
				sum_y = sum_y + ?,
				sum_w = sum_w + ?,
				sum_h = sum_h + ?,
				sum_x2 = sum_x2 + ?,
				sum_y2 = sum_y2 + ?,
				total_frames = total_frames + 1,
				frequency = CAST(sample_count + 1 AS REAL) / CAST(total_frames + 1 AS REAL),
				updated_at = datetime('now')`,
			class, normalized,
			xNorm, yNorm, wNorm, hNorm, xNorm*xNorm, yNorm*yNorm,
			xNorm, yNorm, wNorm, hNorm, xNorm*xNorm, yNorm*yNorm)
		if err != nil {
			return
		}
	}

	if err := tx.Commit(); err != nil {
		return
	}

	loadPriorsFromDB()
}

func UpdatePriorsForNegative(windowTitle string) {
	elementPriors.mu.Lock()
	defer elementPriors.mu.Unlock()

	if trainDB == nil {
		return
	}

	normalized := normalizeWindowTitle(windowTitle)

	_, err := trainDB.Exec(`UPDATE element_priors SET
		total_frames = total_frames + 1,
		frequency = CAST(sample_count AS REAL) / CAST((total_frames + 1) AS REAL),
		updated_at = datetime('now')
		WHERE window_title = ?`, normalized)
	if err != nil {
		return
	}

	loadPriorsFromDB()
}

func AdjustConfidenceWithPriors(className, windowTitle string, confidence float64, x, y float64) float64 {
	if ActiveConfig != nil && !ActiveConfig.PriorAdjustment {
		return confidence
	}

	elementPriors.mu.RLock()
	if !elementPriors.loaded {
		elementPriors.mu.RUnlock()
		loadPriorsFromDB()
		elementPriors.mu.RLock()
	}
	priors := elementPriors.priors
	elementPriors.mu.RUnlock()

	normalized := normalizeWindowTitle(windowTitle)
	var bestPrior *ElementPrior
	for i := range priors {
		if priors[i].Class == className && priors[i].WindowTitle == normalized {
			bestPrior = &priors[i]
			break
		}
	}

	if bestPrior == nil {
		return confidence
	}

	adjusted := confidence

	if bestPrior.Frequency > 0.3 && bestPrior.SampleCount >= 3 {
		boost := 0.05 + bestPrior.Frequency*0.1
		adjusted += boost * (1.0 - adjusted)
	}

	if bestPrior.Frequency < 0.05 && bestPrior.SampleCount >= 2 {
		suppress := 0.1 * (1.0 - bestPrior.Frequency*20)
		adjusted -= suppress * adjusted
	}

	if bestPrior.StdX > 0 && bestPrior.StdY > 0 && bestPrior.SampleCount >= 3 {
		posX := x / 100.0
		posY := y / 100.0
		dzX := (posX - bestPrior.AvgX) / bestPrior.StdX
		dzY := (posY - bestPrior.AvgY) / bestPrior.StdY
		dist := math.Sqrt(dzX*dzX + dzY*dzY)
		if dist > 3.0 {
			adjusted -= 0.15 * adjusted
		} else if dist > 2.0 {
			adjusted -= 0.05 * adjusted
		}
	}

	if adjusted < 0.01 {
		adjusted = 0.01
	}
	if adjusted > 0.99 {
		adjusted = 0.99
	}
	return adjusted
}

type PriorStatsOutput struct {
	Priors       []ElementPrior `json:"priors"`
	TotalEntries int            `json:"total_entries"`
	TotalClasses int            `json:"total_classes"`
}

func GetPriorStats(minCount int) (*PriorStatsOutput, error) {
	if minCount <= 0 {
		minCount = 1
	}

	elementPriors.mu.RLock()
	if !elementPriors.loaded {
		elementPriors.mu.RUnlock()
		loadPriorsFromDB()
		elementPriors.mu.RLock()
	}
	priors := elementPriors.priors
	elementPriors.mu.RUnlock()

	var filtered []ElementPrior
	classes := make(map[string]bool)
	for _, p := range priors {
		if p.SampleCount >= minCount {
			filtered = append(filtered, p)
		}
		classes[p.Class] = true
	}
	if filtered == nil {
		filtered = []ElementPrior{}
	}

	return &PriorStatsOutput{
		Priors:       filtered,
		TotalEntries: len(priors),
		TotalClasses: len(classes),
	}, nil
}

type YoloDatasetStats struct {
	TotalImages int            `json:"total_images"`
	TotalLabels int            `json:"total_labels"`
	ClassCounts map[string]int `json:"class_counts"`
	OutputDir   string         `json:"output_dir"`
}

func ExportYoloDataset(outputDir string, minSignal int) (*YoloDatasetStats, error) {
	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return nil, err
		}
	}

	for _, d := range []string{
		filepath.Join(outputDir, "images", "train"),
		filepath.Join(outputDir, "images", "val"),
		filepath.Join(outputDir, "labels", "train"),
		filepath.Join(outputDir, "labels", "val"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("create %s: %w", d, err)
		}
	}

	minS := minSignal
	if minS <= 0 {
		minS = 1
	}

	rows, err := trainDB.Query(`SELECT id, image_path, onnx_detections, elements_count
		FROM training_samples WHERE signal_level >= ? AND used_for_training = 0
		ORDER BY created_at`, minS)
	if err != nil {
		return nil, fmt.Errorf("query samples: %w", err)
	}
	defer rows.Close()

	classIDMap := make(map[string]int)
	for i, name := range yoloLabels {
		classIDMap[name] = i
	}

	type sample struct {
		ID         int64
		ImagePath  string
		Detections []DetectedElement
		ElemCount  int
	}

	var samples []sample
	for rows.Next() {
		var s sample
		var detJSON string
		if err := rows.Scan(&s.ID, &s.ImagePath, &detJSON, &s.ElemCount); err != nil {
			continue
		}
		json.Unmarshal([]byte(detJSON), &s.Detections)
		if len(s.Detections) == 0 {
			continue
		}
		samples = append(samples, s)
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("no unused samples with signal_level >= %d", minS)
	}

	splitIdx := len(samples) * 80 / 100
	if splitIdx >= len(samples) {
		splitIdx = len(samples) - 1
	}
	if splitIdx < 1 {
		splitIdx = 1
	}

	trainSamples := samples[:splitIdx]
	valSamples := samples[splitIdx:]

	classCounts := make(map[string]int)

	writeSamples := func(samples []sample, subset string) {
		for _, s := range samples {
			imgData, err := os.ReadFile(s.ImagePath)
			if err != nil {
				continue
			}
			base := fmt.Sprintf("sample_%d", s.ID)
			imgDest := filepath.Join(outputDir, "images", subset, base+".png")
			if err := os.WriteFile(imgDest, imgData, 0644); err != nil {
				continue
			}

			img, err := png.Decode(bytes.NewReader(imgData))
			if err != nil {
				continue
			}
			bounds := img.Bounds()
			imgW := float64(bounds.Dx())
			imgH := float64(bounds.Dy())
			if imgW == 0 || imgH == 0 {
				continue
			}

			var labelLines []string
			for _, det := range s.Detections {
				cid, ok := classIDMap[det.Class]
				if !ok {
					continue
				}
				xc := (float64(det.X) + float64(det.W)/2) / imgW
				yc := (float64(det.Y) + float64(det.H)/2) / imgH
				w := float64(det.W) / imgW
				h := float64(det.H) / imgH
				xc = clamp(xc, 0.001, 0.999)
				yc = clamp(yc, 0.001, 0.999)
				w = clamp(w, 0.001, 0.999)
				h = clamp(h, 0.001, 0.999)
				labelLines = append(labelLines, fmt.Sprintf("%d %.6f %.6f %.6f %.6f", cid, xc, yc, w, h))
				classCounts[det.Class]++
			}
			if len(labelLines) == 0 {
				continue
			}
			labelDest := filepath.Join(outputDir, "labels", subset, base+".txt")
			os.WriteFile(labelDest, []byte(strings.Join(labelLines, "\n")), 0644)
		}
	}

	writeSamples(trainSamples, "train")
	writeSamples(valSamples, "val")

	yamlContent := fmt.Sprintf(`# YOLO dataset generated by go-mcp-computer-use
path: %s
train: images/train
val: images/val
nc: %d
names: [%s]
`, outputDir, len(yoloLabels), joinNames(yoloLabels))
	os.WriteFile(filepath.Join(outputDir, "dataset.yaml"), []byte(yamlContent), 0644)

	return &YoloDatasetStats{
		TotalImages: len(samples),
		TotalLabels: len(classCounts),
		ClassCounts: classCounts,
		OutputDir:   outputDir,
	}, nil
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func joinNames(names []string) string {
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(parts, ", ")
}

func TrainingCleanupNoise(maxAgeHours int, dryRun bool) (map[string]int, error) {
	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return nil, err
		}
	}

	result := map[string]int{
		"dry_run":     0,
		"deleted":     0,
		"freed_bytes": 0,
	}
	if dryRun {
		result["dry_run"] = 1
	}

	if maxAgeHours <= 0 {
		maxAgeHours = 24
	}

	rows, err := trainDB.Query(`SELECT id, image_path FROM training_samples
		WHERE signal_level = 0 AND created_at < datetime('now', ?)
		LIMIT 500`, fmt.Sprintf("-%d hours", maxAgeHours))
	if err != nil {
		return nil, fmt.Errorf("query noise: %w", err)
	}
	defer rows.Close()

	var ids []int64
	var paths []string
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}
		ids = append(ids, id)
		paths = append(paths, path)
	}

	if len(ids) == 0 {
		return result, nil
	}

	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			result["freed_bytes"] += int(info.Size())
		}
		if !dryRun {
			os.Remove(path)
		}
	}

	if !dryRun {
		for _, id := range ids {
			trainDB.Exec("DELETE FROM training_samples WHERE id = ?", id)
		}
		result["deleted"] = len(ids)
	}

	return result, nil
}
