package actions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	TrainingSourceRaw     = "raw"
	TrainingSourceWatcher = "watcher"

	TrainingCatClick        = "click"
	TrainingCatType         = "type"
	TrainingCatNavigate     = "navigate"
	TrainingCatOCR          = "ocr"
	TrainingCatGeneral      = "general"
	TrainingCatElementsFound = "elements_found"
	TrainingCatNoElements   = "no_elements"
)

var (
	trainDB     *sql.DB
	trainMu     sync.Mutex
	trainOnce   sync.Once
	trainRoot   string
	rawDir      string
	watcherDir  string
)

var rawCategories = []string{"click", "type", "navigate", "ocr", "general"}
var watcherCategories = []string{"elements_found", "no_elements"}

func getTrainingRoot() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "go-mcp-computer-use", "training")
}

func InitTrainingStore() error {
	var initErr error
	trainOnce.Do(func() {
		trainRoot = getTrainingRoot()
		if trainRoot == "" {
			initErr = fmt.Errorf("APPDATA not set")
			return
		}
		rawDir = filepath.Join(trainRoot, "raw")
		watcherDir = filepath.Join(trainRoot, "watcher")
		for _, cat := range rawCategories {
			if err := os.MkdirAll(filepath.Join(rawDir, cat), 0755); err != nil {
				initErr = fmt.Errorf("create raw/%s: %w", cat, err)
				return
			}
		}
		for _, cat := range watcherCategories {
			if err := os.MkdirAll(filepath.Join(watcherDir, cat), 0755); err != nil {
				initErr = fmt.Errorf("create watcher/%s: %w", cat, err)
				return
			}
		}
		dbPath := filepath.Join(trainRoot, "samples.db")
		db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
		if err != nil {
			initErr = fmt.Errorf("open samples db: %w", err)
			return
		}
		if err := createTrainingTables(db); err != nil {
			db.Close()
			initErr = fmt.Errorf("create training tables: %w", err)
			return
		}
		trainDB = db
	})
	return initErr
}

func createTrainingTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS training_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT 'raw',
			category TEXT NOT NULL,
			image_path TEXT NOT NULL,
			task_prompt TEXT NOT NULL DEFAULT '',
			window_title TEXT NOT NULL DEFAULT '',
			ocr_text TEXT NOT NULL DEFAULT '',
			onnx_detections TEXT NOT NULL DEFAULT '[]',
			elements_count INTEGER NOT NULL DEFAULT 0,
			signal_level INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			used_for_training INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_source ON training_samples(source)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_category ON training_samples(category)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_used ON training_samples(used_for_training)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_signal ON training_samples(signal_level)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	// Migrations for existing databases
	db.Exec("ALTER TABLE training_samples ADD COLUMN signal_level INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE training_samples ADD COLUMN window_rect TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE training_samples ADD COLUMN normalized_coords TEXT NOT NULL DEFAULT '[]'")
	_ = createPriorsTable(db)
	return nil
}

type TrainingSample struct {
	ID                int64               `json:"id"`
	Source            string              `json:"source"`
	Category          string              `json:"category"`
	ImagePath         string              `json:"image_path"`
	TaskPrompt        string              `json:"task_prompt,omitempty"`
	WindowTitle       string              `json:"window_title,omitempty"`
	WindowRect        string              `json:"window_rect,omitempty"`
	OCRText           string              `json:"ocr_text,omitempty"`
	ONNXDetections    []DetectedElement   `json:"onnx_detections,omitempty"`
	NormalizedCoords  []NormalizedElement `json:"normalized_coords,omitempty"`
	ElementsCount     int                 `json:"elements_count"`
	SignalLevel       int                 `json:"signal_level"`
	CreatedAt         string              `json:"created_at"`
	UsedForTraining   bool                `json:"used_for_training"`
}

type SaveTrainingSampleInput struct {
	Source      string `json:"source"`
	Category    string `json:"category"`
	TaskPrompt  string `json:"task_prompt"`
	ImageB64    string `json:"image_b64,omitempty"`
	WindowTitle string `json:"window_title,omitempty"`
	OCRText     string `json:"ocr_text,omitempty"`
}

func imageBaseDir(source string) string {
	switch source {
	case TrainingSourceWatcher:
		return watcherDir
	default:
		return rawDir
	}
}

func saveTrainingSampleDirect(source, category, taskPrompt, imageB64, windowTitle, ocrText string, detections []DetectedElement, normalized []NormalizedElement, windowRect string) (*TrainingSample, error) {
	trainMu.Lock()
	defer trainMu.Unlock()

	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return nil, fmt.Errorf("training store not initialized: %w", err)
		}
	}

	img, err := decodePNGB64(imageB64)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	baseDir := imageBaseDir(source)
	catDir := filepath.Join(baseDir, category)
	if err := os.MkdirAll(catDir, 0755); err != nil {
		return nil, fmt.Errorf("create %s/%s: %w", source, category, err)
	}

	ts := time.Now().UnixMilli()
	fileName := fmt.Sprintf("%d.png", ts)
	imagePath := filepath.Join(catDir, fileName)
	if err := savePNG(imagePath, img); err != nil {
		return nil, fmt.Errorf("save image: %w", err)
	}

	detJSON := "[]"
	detCount := 0
	if len(detections) > 0 {
		b, _ := json.Marshal(detections)
		detJSON = string(b)
		detCount = len(detections)
	}

	signalLevel := computeSignalLevel(detCount, category, taskPrompt)

	normJSON := "[]"
	if len(normalized) > 0 {
		b, _ := json.Marshal(normalized)
		normJSON = string(b)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err = trainDB.Exec(`INSERT INTO training_samples(
		source, category, image_path, task_prompt, window_title, window_rect, ocr_text,
		onnx_detections, normalized_coords, elements_count, signal_level, created_at, used_for_training
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		source, category, imagePath, taskPrompt, windowTitle, windowRect,
		ocrText, detJSON, normJSON, detCount, signalLevel, now)
	if err != nil {
		return nil, fmt.Errorf("insert training sample: %w", err)
	}

	if len(detections) > 0 && windowTitle != "" {
		go UpdatePriorsFromDetections(windowTitle, detections)
	} else if len(detections) == 0 && windowTitle != "" {
		go UpdatePriorsForNegative(windowTitle)
	}

	return &TrainingSample{
		ImagePath:       imagePath,
		Source:          source,
		Category:        category,
		TaskPrompt:      taskPrompt,
		WindowTitle:     windowTitle,
		WindowRect:      windowRect,
		OCRText:         ocrText,
		NormalizedCoords: normalized,
		CreatedAt:       now,
		UsedForTraining: false,
		ElementsCount:   detCount,
		SignalLevel:     signalLevel,
	}, nil
}

func SaveTrainingSample(in SaveTrainingSampleInput) (*TrainingSample, error) {
	var detections []DetectedElement
	var normalized []NormalizedElement
	windowRect := ""
	if detResult, err := ONNXDetect(DetectionInput{ImageB64: in.ImageB64}); err == nil {
		detections = detResult.Elements
		normalized = detResult.Normalized
		if info, err := GetActiveWindowInfo(); err == nil && info != nil && info.Handle != 0 {
			if rect, err := GetWindowRectByHandle(info.Handle); err == nil {
				b, _ := json.Marshal(rect)
				windowRect = string(b)
			}
		}
	}
	return saveTrainingSampleDirect(in.Source, in.Category, in.TaskPrompt, in.ImageB64, in.WindowTitle, in.OCRText, detections, normalized, windowRect)
}

func SaveScreenshotTrainingSample(source, category, taskPrompt, windowTitle, ocrText string) (*TrainingSample, error) {
	b64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("capture screenshot: %w", err)
	}
	return SaveTrainingSample(SaveTrainingSampleInput{
		Source:      source,
		Category:    category,
		TaskPrompt:  taskPrompt,
		ImageB64:    b64,
		WindowTitle: windowTitle,
		OCRText:     ocrText,
	})
}

func computeSignalLevel(detCount int, category, taskPrompt string) int {
	if detCount == 0 {
		return 0
	}
	if category != "" || taskPrompt != "" {
		return 2
	}
	return 1
}

type TrainingListInput struct {
	Source      string `json:"source,omitempty"`
	Category    string `json:"category,omitempty"`
	MinSignal   int    `json:"min_signal,omitempty"`
	UnusedOnly  bool   `json:"unused_only,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type TrainingStats struct {
	TotalSamples  int            `json:"total_samples"`
	UnusedSamples int            `json:"unused_samples"`
	BySource      map[string]int `json:"by_source"`
	ByCategory    map[string]int `json:"by_category"`
	DiskUsageMB   float64        `json:"disk_usage_mb"`
}

func TrainingSampleList(in TrainingListInput) ([]TrainingSample, error) {
	trainMu.Lock()
	defer trainMu.Unlock()

	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return nil, err
		}
	}
	if in.Limit <= 0 || in.Limit > 200 {
		in.Limit = 50
	}

	var where []string
	var args []any
	if in.Source != "" {
		where = append(where, "source = ?")
		args = append(args, in.Source)
	}
	if in.Category != "" {
		where = append(where, "category = ?")
		args = append(args, in.Category)
	}
	if in.MinSignal > 0 {
		where = append(where, "signal_level >= ?")
		args = append(args, in.MinSignal)
	}
	if in.UnusedOnly {
		where = append(where, "used_for_training = 0")
	}

	query := `SELECT id, source, category, image_path, task_prompt, window_title, window_rect,
		ocr_text, onnx_detections, normalized_coords, elements_count, signal_level, created_at, used_for_training
		FROM training_samples`
	if len(where) > 0 {
		query += " WHERE " + joinWhere(where)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, in.Limit)

	rows, err := trainDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list training samples: %w", err)
	}
	defer rows.Close()

	var samples []TrainingSample
	for rows.Next() {
		var s TrainingSample
		var detJSON, normJSON string
		if err := rows.Scan(&s.ID, &s.Source, &s.Category, &s.ImagePath,
			&s.TaskPrompt, &s.WindowTitle, &s.WindowRect, &s.OCRText,
			&detJSON, &normJSON,
			&s.ElementsCount, &s.SignalLevel, &s.CreatedAt, &s.UsedForTraining); err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		json.Unmarshal([]byte(detJSON), &s.ONNXDetections)
		json.Unmarshal([]byte(normJSON), &s.NormalizedCoords)
		samples = append(samples, s)
	}
	return samples, rows.Err()
}

func TrainingStatsReport() (*TrainingStats, error) {
	trainMu.Lock()
	defer trainMu.Unlock()

	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return nil, err
		}
	}

	stats := &TrainingStats{
		BySource:   make(map[string]int),
		ByCategory: make(map[string]int),
	}

	trainDB.QueryRow("SELECT COUNT(*) FROM training_samples").Scan(&stats.TotalSamples)
	trainDB.QueryRow("SELECT COUNT(*) FROM training_samples WHERE used_for_training = 0").Scan(&stats.UnusedSamples)

	rows, err := trainDB.Query("SELECT source, COUNT(*) FROM training_samples GROUP BY source")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var src string
			var count int
			rows.Scan(&src, &count)
			stats.BySource[src] = count
		}
	}

	rows2, err := trainDB.Query("SELECT category, COUNT(*) FROM training_samples GROUP BY category")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var cat string
			var count int
			rows2.Scan(&cat, &count)
			stats.ByCategory[cat] = count
		}
	}

	if trainRoot != "" {
		var totalBytes int64
		for _, d := range []string{rawDir, watcherDir} {
			if d == "" {
				continue
			}
			filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					totalBytes += info.Size()
				}
				return nil
			})
		}
		stats.DiskUsageMB = float64(totalBytes) / (1024 * 1024)
	}

	return stats, nil
}

func TrainingMarkUsed(sampleID int64) error {
	trainMu.Lock()
	defer trainMu.Unlock()

	if trainDB == nil {
		if err := InitTrainingStore(); err != nil {
			return err
		}
	}
	_, err := trainDB.Exec("UPDATE training_samples SET used_for_training = 1 WHERE id = ?", sampleID)
	return err
}

func SaveSnapshotAfterAction(source, category, taskPrompt string) {
	if ActiveConfig != nil && !ActiveConfig.TrainingEnabled {
		return
	}
	go func() {
		SaveScreenshotTrainingSample(source, category, taskPrompt, "", "")
	}()
}

func joinWhere(conditions []string) string {
	if len(conditions) == 0 {
		return "1=1"
	}
	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result += " AND " + conditions[i]
	}
	return result
}
