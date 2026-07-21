package tokenizer

// Tokenizer converts OCR text into integer token IDs for the transformer.
type Tokenizer interface {
	// Fit builds the vocabulary from a corpus of OCR text strings.
	Fit(corpus []string) error

	// Encode converts text into a fixed-length token ID sequence.
	// Unknown words are mapped to the [UNK] token.
	// Output is padded/truncated to maxLen.
	Encode(text string, maxLen int) []int

	// Decode converts token IDs back to text.
	Decode(tokens []int) string

	// VocabSize returns the number of tokens in the vocabulary.
	VocabSize() int

	// Save writes the vocabulary to a file path.
	Save(path string) error

	// Load reads the vocabulary from a file path.
	Load(path string) error
}

// DefaultMaxLen is the default maximum sequence length for token encoding.
const DefaultMaxLen = 128
