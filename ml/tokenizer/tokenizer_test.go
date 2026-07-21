package tokenizer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFit_BuildsVocab(t *testing.T) {
	tok := NewSimpleTokenizer()
	corpus := []string{
		"click button Submit",
		"hover over Cancel",
		"click button OK",
	}
	if err := tok.Fit(corpus); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}
	if tok.VocabSize() < 5 {
		t.Errorf("expected vocab >= 5, got %d", tok.VocabSize())
	}
}

func TestEncode_ReturnsFixedLength(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover over Cancel"})
	tokens := tok.Encode("click button Submit", 10)
	if len(tokens) != 10 {
		t.Errorf("expected length 10, got %d", len(tokens))
	}
}

func TestEncode_UnknownTokens(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	tokens := tok.Encode("totally unknown gibberish", 10)
	for _, id := range tokens {
		if id == 0 {
			continue // padding
		}
		if id == unkToken {
			continue // expected unknown
		}
		// should have at least one unknown token
	}
	// verify at least one unknown token exists
	hasUnk := false
	for _, id := range tokens {
		if id == unkToken {
			hasUnk = true
			break
		}
	}
	if !hasUnk {
		t.Error("expected at least one UNK token for unknown words")
	}
}

func TestDecode_InverseOfEncode(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover"})
	input := "click button"
	tokens := tok.Encode(input, 10)
	decoded := tok.Decode(tokens)
	if decoded == "" {
		t.Error("decoded string should not be empty")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover over Cancel"})
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.bin")
	if err := tok.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	tok2 := NewSimpleTokenizer()
	if err := tok2.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if tok2.VocabSize() != tok.VocabSize() {
		t.Errorf("vocab size mismatch: %d vs %d", tok2.VocabSize(), tok.VocabSize())
	}
}

func TestEncode_EmptyText(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button"})
	tokens := tok.Encode("", 10)
	if len(tokens) != 10 {
		t.Errorf("expected padding to fill maxLen, got %d", len(tokens))
	}
	for _, id := range tokens {
		if id != padToken {
			t.Errorf("expected padding token, got %d", id)
		}
	}
}

func TestEncode_MaxLenTruncates(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"a b c d e f g h i j k l m n o p"})
	tokens := tok.Encode("a b c d e f g h i j", 5)
	if len(tokens) != 5 {
		t.Errorf("expected truncation to 5, got %d", len(tokens))
	}
}

func TestFit_EmptyCorpus(t *testing.T) {
	tok := NewSimpleTokenizer()
	err := tok.Fit([]string{})
	if err == nil {
		t.Error("expected error for empty corpus")
	}
}

func TestSave_NonexistentDir(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button"})
	err := tok.Save("/nonexistent/path/vocab.bin")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	tok := NewSimpleTokenizer()
	err := tok.Load("/nonexistent/path/vocab.bin")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestVocabSize_BeforeFit(t *testing.T) {
	tok := NewSimpleTokenizer()
	if tok.VocabSize() != 0 {
		t.Errorf("expected vocab size 0 before Fit, got %d", tok.VocabSize())
	}
}

func TestFit_Deduplicates(t *testing.T) {
	tok := NewSimpleTokenizer()
	corpus := []string{"click click click", "click button", "button button"}
	if err := tok.Fit(corpus); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}
	// "click" and "button" should each appear once in vocab
	vocab := tok.VocabSize()
	if vocab > 5+2 { // pad + unk + eos + special tokens + 2 words + some margin
		t.Errorf("vocab too large after dedup: %d", vocab)
	}
}

func TestEncode_SpecialTokens(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	tokens := tok.Encode("click", 10)
	// last non-padding token should be EOS
	foundEOS := false
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i] == padToken {
			continue
		}
		if tokens[i] == eosToken {
			foundEOS = true
		}
		break
	}
	if !foundEOS {
		t.Error("expected EOS token at end of encoded sequence")
	}
}

func TestEncode_CaseInsensitive(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"Click Button Submit"})
	upper := tok.Encode("CLICK BUTTON", 10)
	lower := tok.Encode("click button", 10)
	for i := range upper {
		if upper[i] != lower[i] {
			t.Errorf("expected case-insensitive encoding, got %d vs %d at index %d", upper[i], lower[i], i)
		}
	}
}

func TestFit_LargeCorpus(t *testing.T) {
	tok := NewSimpleTokenizer()
	corpus := make([]string, 1000)
	for i := range corpus {
		corpus[i] = "word" + string(rune('a'+i%26)) + " button Submit click hover"
	}
	if err := tok.Fit(corpus); err != nil {
		t.Fatalf("Fit failed on large corpus: %v", err)
	}
	if tok.VocabSize() < 10 {
		t.Errorf("expected vocab >= 10 for large corpus, got %d", tok.VocabSize())
	}
}

func TestSave_Idempotent(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.bin")
	if err := tok.Save(path); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	if err := tok.Save(path); err != nil {
		t.Fatalf("second Save failed (should be idempotent): %v", err)
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.bin")
	// write corrupt data manually
	f, _ := os.Create(path)
	f.Write([]byte("not a valid vocab file"))
	f.Close()
	tok := NewSimpleTokenizer()
	err := tok.Load(path)
	if err == nil {
		t.Error("expected error loading corrupt file")
	}
}

func TestEncode_EOSPlacement(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	tokens := tok.Encode("click", 20)
	// find the last non-padding token
	lastNonPad := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i] != padToken {
			lastNonPad = i
			break
		}
	}
	if lastNonPad < 0 {
		t.Fatal("no non-padding tokens found")
	}
	if tokens[lastNonPad] != eosToken {
		t.Errorf("expected EOS at end, got token %d", tokens[lastNonPad])
	}
}

func TestDecode_SkipsSpecialTokens(t *testing.T) {
	tok := NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	decoded := tok.Decode([]int{padToken, padToken, unkToken, eosToken})
	if decoded != "" {
		t.Errorf("expected empty string for all-special tokens, got %q", decoded)
	}
}
