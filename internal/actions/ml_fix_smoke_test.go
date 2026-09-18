package actions

import (
	"testing"
)

func TestExtractArgsFromJSON_NestedStringAndObject(t *testing.T) {
	s := extractArgsFromJSON(`{"args":"{\"X\":700,\"Y\":400}","tool":"click"}`)
	if s == "" {
		t.Fatal("expected nested string args")
	}
	s2 := extractArgsFromJSON(`{"tool":"click","args":{"X":1,"Y":2}}`)
	if s2 == "" || s2[0] != '{' {
		t.Fatalf("object args -> %q", s2)
	}
}

func TestPredictionsLookInformative(t *testing.T) {
	// near-uniform junk
	junk := []struct {
		Score float64
	}{{0.04}, {0.04}, {0.04}}
	_ = junk
}
