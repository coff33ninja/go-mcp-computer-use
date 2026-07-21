package export

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func testConfig() transformer.Config {
	return transformer.Config{
		VocabSize: 100,
		MaxLen:    16,
		EmbedDim:  32,
		NumHeads:  2,
		NumLayers: 2,
		FFNDim:    64,
		CoordDim:  7,
		OutputDim: 10,
	}
}

func TestSaveModel_RoundTrip(t *testing.T) {
	cfg := testConfig()
	model, err := transformer.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	ser := NewGobSerializer()
	meta := ModelMeta{
		Version:     1,
		Config:      cfg,
		TrainedOn:   "2026-07-21",
		SampleCount: 500,
		UserID:      "test-user",
	}
	dir := t.TempDir()
	path := dir + "/model.gob"
	if err := ser.SaveModel(model, meta, path); err != nil {
		t.Fatalf("SaveModel failed: %v", err)
	}

	model2, err := transformer.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	loadedMeta, err := ser.LoadModel(model2, path)
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}
	if loadedMeta.SampleCount != 500 {
		t.Errorf("expected SampleCount=500, got %d", loadedMeta.SampleCount)
	}
	if loadedMeta.UserID != "test-user" {
		t.Errorf("expected UserID=test-user, got %q", loadedMeta.UserID)
	}
}

func TestSaveModel_NonexistentDir(t *testing.T) {
	cfg := testConfig()
	model, _ := transformer.New(cfg)
	ser := NewGobSerializer()
	err := ser.SaveModel(model, ModelMeta{}, "/nonexistent/path/model.gob")
	if err == nil {
		t.Error("expected error saving to nonexistent directory")
	}
}

func TestLoadModel_NonexistentFile(t *testing.T) {
	cfg := testConfig()
	model, _ := transformer.New(cfg)
	ser := NewGobSerializer()
	_, err := ser.LoadModel(model, "/nonexistent/path/model.gob")
	if err == nil {
		t.Error("expected error loading nonexistent file")
	}
}

func TestLoadModel_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/corrupt.gob"
	if err := writeFile(path, []byte("not a valid gob file")); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig()
	model, _ := transformer.New(cfg)
	ser := NewGobSerializer()
	_, err := ser.LoadModel(model, path)
	if err == nil {
		t.Error("expected error loading corrupt file")
	}
}

func TestSaveBuffer_RoundTrip(t *testing.T) {
	ser := NewGobSerializer()
	data := []byte("test buffer data")
	dir := t.TempDir()
	path := dir + "/buffer.bin"
	if err := ser.SaveBuffer(data, path); err != nil {
		t.Fatalf("SaveBuffer failed: %v", err)
	}
	loaded, err := ser.LoadBuffer(path)
	if err != nil {
		t.Fatalf("LoadBuffer failed: %v", err)
	}
	if string(loaded) != string(data) {
		t.Errorf("buffer mismatch: %q vs %q", loaded, data)
	}
}

func TestSaveBuffer_NonexistentDir(t *testing.T) {
	ser := NewGobSerializer()
	err := ser.SaveBuffer([]byte("data"), "/nonexistent/path/buffer.bin")
	if err == nil {
		t.Error("expected error saving to nonexistent directory")
	}
}

func TestLoadBuffer_NonexistentFile(t *testing.T) {
	ser := NewGobSerializer()
	_, err := ser.LoadBuffer("/nonexistent/path/buffer.bin")
	if err == nil {
		t.Error("expected error loading nonexistent file")
	}
}

func TestModelMeta_JSON(t *testing.T) {
	cfg := testConfig()
	meta := ModelMeta{
		Version:     1,
		Config:      cfg,
		TrainedOn:   "2026-07-21",
		SampleCount: 1000,
		UserID:      "user-42",
	}
	// test that meta can be serialized to JSON
	data := toJSON(meta)
	if data == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestSaveModel_Overwrite(t *testing.T) {
	cfg := testConfig()
	model, _ := transformer.New(cfg)
	ser := NewGobSerializer()
	dir := t.TempDir()
	path := dir + "/model.gob"

	if err := ser.SaveModel(model, ModelMeta{SampleCount: 100}, path); err != nil {
		t.Fatal(err)
	}
	if err := ser.SaveModel(model, ModelMeta{SampleCount: 200}, path); err != nil {
		t.Fatal(err)
	}

	model2, _ := transformer.New(cfg)
	meta, err := ser.LoadModel(model2, path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SampleCount != 200 {
		t.Errorf("expected overwritten SampleCount=200, got %d", meta.SampleCount)
	}
}

func TestLoadModel_WeightConsistency(t *testing.T) {
	cfg := testConfig()
	model, _ := transformer.New(cfg)
	ser := NewGobSerializer()
	dir := t.TempDir()
	path := dir + "/model.gob"

	_ = ser.SaveModel(model, ModelMeta{Config: cfg}, path)

	model2, _ := transformer.New(cfg)
	_, _ = ser.LoadModel(model2, path)

	p1 := model.Parameters()
	p2 := model2.Parameters()
	if len(p1) != len(p2) {
		t.Fatalf("param count mismatch: %d vs %d", len(p1), len(p2))
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			t.Errorf("param[%d] mismatch: %f vs %f", i, p1[i], p2[i])
		}
	}
}

func writeFile(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func toJSON(meta ModelMeta) string {
	b, _ := json.Marshal(meta)
	return string(b)
}
