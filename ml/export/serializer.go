package export

import "github.com/coff33ninja/go-mcp-computer-use/ml/transformer"

// ModelMeta stores metadata alongside the trained weights.
type ModelMeta struct {
	Version     int                `json:"version"`
	Config      transformer.Config `json:"config"`
	TrainedOn   string             `json:"trained_on"`   // timestamp
	SampleCount int                `json:"sample_count"` // number of training samples
	UserID      string             `json:"user_id"`      // user identifier
}

// Serializer handles saving and loading trained model checkpoints.
type Serializer interface {
	// SaveModel writes the model weights and metadata to a file.
	SaveModel(model transformer.Model, meta ModelMeta, path string) error

	// LoadModel reads the model weights and metadata from a file.
	LoadModel(model transformer.Model, path string) (*ModelMeta, error)

	// SaveBuffer writes the experience replay buffer to a file.
	SaveBuffer(data []byte, path string) error

	// LoadBuffer reads the experience replay buffer from a file.
	LoadBuffer(path string) ([]byte, error)
}
