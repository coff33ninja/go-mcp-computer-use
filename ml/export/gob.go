package export

import (
	"encoding/gob"
	"os"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// GobSerializer implements Serializer using Go's encoding/gob format.
type GobSerializer struct{}

// NewGobSerializer creates a new gob-based serializer.
func NewGobSerializer() *GobSerializer {
	return &GobSerializer{}
}

func (s *GobSerializer) SaveModel(model transformer.Model, meta ModelMeta, path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := gob.NewEncoder(f).Encode(struct {
		Meta   ModelMeta
		Params []float64
	}{Meta: meta, Params: model.Parameters()}); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (s *GobSerializer) LoadModel(model transformer.Model, path string) (*ModelMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data struct {
		Meta   ModelMeta
		Params []float64
	}
	if err := gob.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	if err := model.LoadParameters(data.Params); err != nil {
		return nil, err
	}
	return &data.Meta, nil
}

func (s *GobSerializer) SaveBuffer(data []byte, path string) error {
	return os.WriteFile(path, data, 0644)
}

func (s *GobSerializer) LoadBuffer(path string) ([]byte, error) {
	return os.ReadFile(path)
}
