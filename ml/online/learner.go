package online

import (
	"encoding/gob"
	"os"
	"sync"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// Experience represents a single training interaction.
type Experience struct {
	Context     string  // OCR text at time of action
	Action      string  // tool name
	ArgsJSON    string  // tool arguments
	Success     bool    // outcome
	CoordX      int     // X coordinate
	CoordY      int     // Y coordinate
	WindowTitle string  // window title
}

// Learner provides online learning via experience replay.
type Learner interface {
	// Store adds an experience to the replay buffer.
	Store(exp Experience) error

	// TrainOnBatch samples a batch from the buffer and updates the model.
	// Returns the mean loss for the batch.
	TrainOnBatch(model transformer.Model, batchSize int, lr float64) (float64, error)

	// BufferSize returns the current number of experiences in the buffer.
	BufferSize() int

	// Save persists the replay buffer to disk.
	Save(path string) error

	// Load restores the replay buffer from disk.
	Load(path string) error
}

// Config configures the online learner.
type Config struct {
	MaxBufferSize int     // maximum experiences to retain (default 10000)
	BatchSize     int     // samples per training step (default 32)
	LearningRate  float64 // SGD learning rate (default 0.001)
}

// DefaultConfig returns sensible defaults for online learning.
func DefaultConfig() Config {
	return Config{
		MaxBufferSize: 10000,
		BatchSize:     32,
		LearningRate:  0.001,
	}
}

// ReplayBuffer stores experiences for online learning.
type ReplayBuffer struct {
	capacity int
	exps     []Experience
	mu       sync.Mutex
}

// NewReplayBuffer creates a buffer with the given maximum capacity.
func NewReplayBuffer(capacity int) *ReplayBuffer {
	return &ReplayBuffer{
		capacity: capacity,
		exps:     make([]Experience, 0, capacity),
	}
}

func (b *ReplayBuffer) Store(exp Experience) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.capacity == 0 {
		return
	}
	if len(b.exps) >= b.capacity {
		b.exps = b.exps[1:]
	}
	b.exps = append(b.exps, exp)
}

func (b *ReplayBuffer) Sample(n int) []Experience {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n <= 0 {
		return nil
	}
	if n > len(b.exps) {
		n = len(b.exps)
	}
	out := make([]Experience, n)
	copy(out, b.exps[:n])
	return out
}

func (b *ReplayBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.exps)
}

func (b *ReplayBuffer) Capacity() int { return b.capacity }

// Save persists the buffer to disk using gob encoding.
func (b *ReplayBuffer) Save(path string) error {
	b.mu.Lock()
	exps := make([]Experience, len(b.exps))
	copy(exps, b.exps)
	b.mu.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(exps)
}

// Load restores the buffer from disk.
func (b *ReplayBuffer) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var exps []Experience
	if err := gob.NewDecoder(f).Decode(&exps); err != nil {
		return err
	}
	b.mu.Lock()
	b.exps = exps
	b.mu.Unlock()
	return nil
}
