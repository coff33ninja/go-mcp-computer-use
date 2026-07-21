package versioning

import (
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// Version tracks a single model checkpoint.
type Version struct {
	ID        int       `json:"id"`
	Loss      float64   `json:"loss"`
	Accuracy  float64   `json:"accuracy"` // rolling accuracy %
	Samples   int       `json:"samples"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
}

// ModelVersion manages versioned model checkpoints with auto-rollback.
type ModelVersion struct {
	mu            sync.RWMutex
	versions      []Version
	nextID        int
	threshold     float64 // max allowed accuracy regression before rollback (0.05 = 5%)
	rollbacks     int
	dataDir       string
}

// New creates a ModelVersion manager for the given data directory.
func New(dataDir string) *ModelVersion {
	mv := &ModelVersion{
		threshold: 0.05,
		dataDir:   dataDir,
	}
	mv.loadIndex()
	return mv
}

// SetThreshold sets the maximum allowed accuracy regression before auto-rollback.
// Default is 0.05 (5%). E.g., 0.05 means if new model is >5% worse, rollback.
func (mv *ModelVersion) SetThreshold(threshold float64) {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.threshold = threshold
}

// SaveCheckpoint saves the current model and records a version.
// Returns the version ID. If model is nil, only records the version without saving.
func (mv *ModelVersion) SaveCheckpoint(model transformer.Model, loss float64, accuracy float64, samples int) (int, error) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	id := mv.nextID
	mv.nextID++

	path := filepath.Join(mv.dataDir, fmt.Sprintf("model_v%d.gob", id))
	if model != nil {
		if err := model.Save(path); err != nil {
			return 0, fmt.Errorf("versioning: save checkpoint v%d: %w", id, err)
		}
	}

	v := Version{
		ID:        id,
		Loss:      loss,
		Accuracy:  accuracy,
		Samples:   samples,
		Timestamp: time.Now(),
		Path:      path,
	}
	mv.versions = append(mv.versions, v)
	mv.saveIndex()

	slog.Info("ml: saved checkpoint",
		"version", id,
		"loss", fmt.Sprintf("%.4f", loss),
		"accuracy", fmt.Sprintf("%.2f%%", accuracy*100),
	)
	return id, nil
}

// CheckAndRollback checks if the new model is worse than the best recent model.
// If regression exceeds threshold, returns the path to the best model to restore.
// Returns empty string if no rollback needed.
func (mv *ModelVersion) CheckAndRollback(currentLoss float64, currentAccuracy float64) string {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.versions) < 2 {
		return ""
	}

	// find best model by accuracy (from last 5 versions)
	recent := mv.versions
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	bestIdx := 0
	for i, v := range recent {
		if v.Accuracy > recent[bestIdx].Accuracy {
			bestIdx = i
		}
	}
	best := recent[bestIdx]

	// check regression
	if best.Accuracy > 0 && currentAccuracy < best.Accuracy-mv.threshold {
		mv.rollbacks++
		slog.Warn("ml: model regression detected, rolling back",
			"current_accuracy", fmt.Sprintf("%.2f%%", currentAccuracy*100),
			"best_accuracy", fmt.Sprintf("%.2f%%", best.Accuracy*100),
			"threshold", fmt.Sprintf("%.2f%%", mv.threshold*100),
			"rollback_count", mv.rollbacks,
		)
		return best.Path
	}
	return ""
}

// Latest returns the most recent version, or nil if none exist.
func (mv *ModelVersion) Latest() *Version {
	mv.mu.RLock()
	defer mv.mu.RUnlock()
	if len(mv.versions) == 0 {
		return nil
	}
	return &mv.versions[len(mv.versions)-1]
}

// Best returns the version with the highest accuracy.
func (mv *ModelVersion) Best() *Version {
	mv.mu.RLock()
	defer mv.mu.RUnlock()
	if len(mv.versions) == 0 {
		return nil
	}
	best := &mv.versions[0]
	for i := range mv.versions {
		if mv.versions[i].Accuracy > best.Accuracy {
			best = &mv.versions[i]
		}
	}
	return best
}

// Rollbacks returns the number of times rollback has been triggered.
func (mv *ModelVersion) Rollbacks() int {
	mv.mu.RLock()
	defer mv.mu.RUnlock()
	return mv.rollbacks
}

// List returns all versions sorted by ID.
func (mv *ModelVersion) List() []Version {
	mv.mu.RLock()
	defer mv.mu.RUnlock()
	out := make([]Version, len(mv.versions))
	copy(out, mv.versions)
	return out
}

func (mv *ModelVersion) indexPath() string {
	return filepath.Join(mv.dataDir, "model_versions.gob")
}

func (mv *ModelVersion) saveIndex() {
	f, err := os.Create(mv.indexPath())
	if err != nil {
		slog.Warn("ml: failed to save version index", "err", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(struct {
		Versions []Version
		NextID   int
		Rollbacks int
	}{mv.versions, mv.nextID, mv.rollbacks})
}

func (mv *ModelVersion) loadIndex() {
	f, err := os.Open(mv.indexPath())
	if err != nil {
		return
	}
	defer f.Close()
	var data struct {
		Versions []Version
		NextID   int
		Rollbacks int
	}
	if err := gob.NewDecoder(f).Decode(&data); err != nil {
		return
	}
	mv.versions = data.Versions
	mv.nextID = data.NextID
	mv.rollbacks = data.Rollbacks
}

// CleanupOldVersions removes old checkpoints, keeping the last N versions.
func (mv *ModelVersion) CleanupOldVersions(keepLast int) int {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	if len(mv.versions) <= keepLast {
		return 0
	}

	sort.Slice(mv.versions, func(i, j int) bool {
		return mv.versions[i].ID < mv.versions[j].ID
	})

	removed := 0
	toRemove := mv.versions[:len(mv.versions)-keepLast]
	for _, v := range toRemove {
		if err := os.Remove(v.Path); err == nil {
			removed++
		}
	}
	mv.versions = mv.versions[len(mv.versions)-keepLast:]
	mv.saveIndex()
	return removed
}
