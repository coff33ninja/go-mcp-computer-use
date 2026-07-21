package tokenizer

import (
	"encoding/gob"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	padToken = 0
	unkToken = 1
	eosToken = 2
)

type vocabEntry struct {
	Word  string
	Count int
}

type SimpleTokenizer struct {
	word2id  map[string]int
	id2word  map[int]string
	bigram2id map[string]int
	fitted   bool
}

func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{
		word2id:  make(map[string]int),
		id2word:  make(map[int]string),
		bigram2id: make(map[string]int),
	}
}

var reMultiSpace = regexp.MustCompile(`\s{2,}`)
var reDigits = regexp.MustCompile(`^\d+$`)
var reAlphaNum = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// normalizeOCR cleans common OCR artifacts before tokenization.
func normalizeOCR(text string) string {
	// remove pipe chars commonly misread from border lines
	text = strings.ReplaceAll(text, "|", "")
	text = strings.ReplaceAll(text, "│", "")
	// collapse multiple spaces
	text = reMultiSpace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (t *SimpleTokenizer) Fit(corpus []string) error {
	if len(corpus) == 0 {
		return fmt.Errorf("tokenizer: empty corpus")
	}

	wordCounts := make(map[string]int)
	bigramCounts := make(map[string]int)

	for _, text := range corpus {
		text = normalizeOCR(text)
		words := strings.Fields(strings.ToLower(text))
		for _, w := range words {
			w = strings.Trim(w, ".,:;!?\"'()[]{}/\\<>@#$%^&*+=~`|")
			if len(w) == 0 {
				continue
			}
			wordCounts[w]++
			// collect character bigrams for subword fallback
			for i := 0; i+1 < len(w); i++ {
				bigram := w[i : i+2]
				bigramCounts[bigram]++
			}
		}
	}

	entries := make([]vocabEntry, 0, len(wordCounts))
	for w, c := range wordCounts {
		entries = append(entries, vocabEntry{Word: w, Count: c})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	// reserve special tokens
	t.word2id = map[string]int{
		"[PAD]": padToken,
		"[UNK]": unkToken,
		"[EOS]": eosToken,
		"[NUM]": 3, // special numeric token
	}
	t.id2word = map[int]string{
		padToken: "[PAD]",
		unkToken: "[UNK]",
		eosToken: "[EOS]",
		3:        "[NUM]",
	}
	id := 4
	for _, e := range entries {
		if _, exists := t.word2id[e.Word]; !exists {
			t.word2id[e.Word] = id
			t.id2word[id] = e.Word
			id++
		}
	}

	// build bigram vocabulary starting after word vocab
	bigramEntries := make([]vocabEntry, 0, len(bigramCounts))
	for bg, c := range bigramCounts {
		bigramEntries = append(bigramEntries, vocabEntry{Word: bg, Count: c})
	}
	sort.Slice(bigramEntries, func(i, j int) bool {
		return bigramEntries[i].Count > bigramEntries[j].Count
	})
	// cap bigram vocab at 512 most frequent
	for i, e := range bigramEntries {
		if i >= 512 {
			break
		}
		t.bigram2id[e.Word] = id
		t.id2word[id] = e.Word
		id++
	}

	t.fitted = true
	return nil
}

// subwordEncode encodes an OOV word using character bigrams.
// Returns the token IDs for the bigrams.
func (t *SimpleTokenizer) subwordEncode(word string) []int {
	if len(word) < 2 {
		return []int{unkToken}
	}
	var ids []int
	for i := 0; i+1 < len(word); i++ {
		bigram := word[i : i+2]
		if id, ok := t.bigram2id[bigram]; ok {
			ids = append(ids, id)
		} else {
			ids = append(ids, unkToken)
		}
	}
	if len(ids) == 0 {
		return []int{unkToken}
	}
	return ids
}

func (t *SimpleTokenizer) Encode(text string, maxLen int) []int {
	if !t.fitted {
		return nil
	}
	text = normalizeOCR(text)
	words := strings.Fields(strings.ToLower(text))
	tokens := make([]int, 0, maxLen)
	for _, w := range words {
		w = strings.Trim(w, ".,:;!?\"'()[]{}/\\<>@#$%^&*+=~`|")
		if len(w) == 0 {
			continue
		}
		// numeric tokens get a special ID
		if reDigits.MatchString(w) {
			tokens = append(tokens, 3) // [NUM]
			if len(tokens) >= maxLen-1 {
				break
			}
			continue
		}
		id, ok := t.word2id[w]
		if ok {
			tokens = append(tokens, id)
		} else {
			// OOV: try bigram subword fallback
			subIDs := t.subwordEncode(w)
			for _, sid := range subIDs {
				tokens = append(tokens, sid)
				if len(tokens) >= maxLen-1 {
					break
				}
			}
		}
		if len(tokens) >= maxLen-1 {
			break
		}
	}
	if len(tokens) > 0 {
		tokens = append(tokens, eosToken)
	}
	for len(tokens) < maxLen {
		tokens = append(tokens, padToken)
	}
	return tokens[:maxLen]
}

func (t *SimpleTokenizer) Decode(tokens []int) string {
	var words []string
	for _, id := range tokens {
		if id == padToken || id == eosToken || id == unkToken {
			continue
		}
		word, ok := t.id2word[id]
		if !ok {
			continue
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

func (t *SimpleTokenizer) VocabSize() int {
	return len(t.word2id)
}

func (t *SimpleTokenizer) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(struct {
		Word2id   map[string]int
		Bigram2id map[string]int
	}{t.word2id, t.bigram2id})
}

func (t *SimpleTokenizer) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// try new format first (with bigrams)
	var data struct {
		Word2id   map[string]int
		Bigram2id map[string]int
	}
	if err := gob.NewDecoder(f).Decode(&data); err == nil && data.Word2id != nil {
		t.word2id = data.Word2id
		t.bigram2id = data.Bigram2id
		if t.bigram2id == nil {
			t.bigram2id = make(map[string]int)
		}
	} else {
		// fall back to old format (word2id only)
		f.Seek(0, 0)
		word2id := make(map[string]int)
		if err := gob.NewDecoder(f).Decode(&word2id); err != nil {
			return err
		}
		t.word2id = word2id
		t.bigram2id = make(map[string]int)
	}
	t.id2word = make(map[int]string, len(t.word2id))
	for w, id := range t.word2id {
		t.id2word[id] = w
	}
	for bg, id := range t.bigram2id {
		t.id2word[id] = bg
	}
	t.fitted = true
	return nil
}

// HasWord returns true if the word exists in the vocabulary.
func (t *SimpleTokenizer) HasWord(word string) bool {
	_, ok := t.word2id[strings.ToLower(word)]
	return ok
}

// IsNumeric returns true if the token is a numeric value.
func IsNumeric(s string) bool {
	return reDigits.MatchString(s)
}

// IsAlphaNum returns true if the string contains only alphanumeric chars.
func IsAlphaNum(s string) bool {
	return reAlphaNum.MatchString(s)
}

// SplitCamelCase splits camelCase/PascalCase words into components.
// e.g. "submitButton" → ["submit", "Button"]
func SplitCamelCase(s string) []string {
	var result []string
	var current strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 && unicode.IsLower(rune(s[i-1])) {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}
