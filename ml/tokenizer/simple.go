package tokenizer

import (
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"strings"
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
	word2id map[string]int
	id2word map[int]string
	fitted  bool
}

func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{
		word2id: make(map[string]int),
		id2word: make(map[int]string),
	}
}

func (t *SimpleTokenizer) Fit(corpus []string) error {
	if len(corpus) == 0 {
		return fmt.Errorf("tokenizer: empty corpus")
	}

	wordCounts := make(map[string]int)
	for _, text := range corpus {
		words := strings.Fields(strings.ToLower(text))
		for _, w := range words {
			w = strings.Trim(w, ".,:;!?\"'()[]{}/\\<>@#$%^&*+=~`|")
			if len(w) > 0 {
				wordCounts[w]++
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
	}
	t.id2word = map[int]string{
		padToken: "[PAD]",
		unkToken: "[UNK]",
		eosToken: "[EOS]",
	}
	id := 3
	for _, e := range entries {
		if _, exists := t.word2id[e.Word]; !exists {
			t.word2id[e.Word] = id
			t.id2word[id] = e.Word
			id++
		}
	}
	t.fitted = true
	return nil
}

func (t *SimpleTokenizer) Encode(text string, maxLen int) []int {
	if !t.fitted {
		return nil
	}
	words := strings.Fields(strings.ToLower(text))
	tokens := make([]int, 0, maxLen)
	for _, w := range words {
		w = strings.Trim(w, ".,:;!?\"'()[]{}/\\<>@#$%^&*+=~`|")
		if len(w) == 0 {
			continue
		}
		id, ok := t.word2id[w]
		if !ok {
			id = unkToken
		}
		tokens = append(tokens, id)
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
	return gob.NewEncoder(f).Encode(t.word2id)
}

func (t *SimpleTokenizer) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	word2id := make(map[string]int)
	if err := gob.NewDecoder(f).Decode(&word2id); err != nil {
		return err
	}
	t.word2id = word2id
	t.id2word = make(map[int]string, len(word2id))
	for w, id := range word2id {
		t.id2word[id] = w
	}
	t.fitted = true
	return nil
}
