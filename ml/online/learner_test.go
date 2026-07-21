package online

import (
	"testing"
)

func TestNewReplayBuffer_Capacity(t *testing.T) {
	buf := NewReplayBuffer(100)
	if buf.Capacity() != 100 {
		t.Errorf("expected capacity 100, got %d", buf.Capacity())
	}
}

func TestStore_IncrementsCount(t *testing.T) {
	buf := NewReplayBuffer(100)
	buf.Store(Experience{Action: "click", Success: true})
	buf.Store(Experience{Action: "hover", Success: false})
	if buf.Size() != 2 {
		t.Errorf("expected size 2, got %d", buf.Size())
	}
}

func TestStore_OverflowDropsOldest(t *testing.T) {
	buf := NewReplayBuffer(3)
	buf.Store(Experience{Action: "a"})
	buf.Store(Experience{Action: "b"})
	buf.Store(Experience{Action: "c"})
	buf.Store(Experience{Action: "d"}) // should drop "a"
	if buf.Size() != 3 {
		t.Errorf("expected size 3 after overflow, got %d", buf.Size())
	}
	exps := buf.Sample(3)
	for _, e := range exps {
		if e.Action == "a" {
			t.Error("expected oldest experience to be dropped")
		}
	}
}

func TestSample_ReturnsCorrectCount(t *testing.T) {
	buf := NewReplayBuffer(100)
	for i := 0; i < 50; i++ {
		buf.Store(Experience{Action: "click"})
	}
	samples := buf.Sample(10)
	if len(samples) != 10 {
		t.Errorf("expected 10 samples, got %d", len(samples))
	}
}

func TestSample_CappedAtBufferSize(t *testing.T) {
	buf := NewReplayBuffer(5)
	for i := 0; i < 10; i++ {
		buf.Store(Experience{Action: "click"})
	}
	samples := buf.Sample(20)
	if len(samples) != 5 {
		t.Errorf("expected min(20,5)=5 samples, got %d", len(samples))
	}
}

func TestSample_ZeroReturnsEmpty(t *testing.T) {
	buf := NewReplayBuffer(100)
	buf.Store(Experience{Action: "click"})
	samples := buf.Sample(0)
	if len(samples) != 0 {
		t.Errorf("expected 0 samples for n=0, got %d", len(samples))
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	buf := NewReplayBuffer(100)
	buf.Store(Experience{Action: "click", Success: true, CoordX: 100, CoordY: 200})
	buf.Store(Experience{Action: "hover", Success: false, CoordX: 50, CoordY: 50})

	dir := t.TempDir()
	path := dir + "/buffer.bin"
	if err := buf.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	buf2 := NewReplayBuffer(100)
	if err := buf2.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if buf2.Size() != 2 {
		t.Errorf("expected 2 loaded experiences, got %d", buf2.Size())
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	buf := NewReplayBuffer(100)
	err := buf.Load("/nonexistent/path/buffer.bin")
	if err == nil {
		t.Error("expected error loading nonexistent file")
	}
}

func TestSave_NonexistentDir(t *testing.T) {
	buf := NewReplayBuffer(100)
	err := buf.Save("/nonexistent/path/buffer.bin")
	if err == nil {
		t.Error("expected error saving to nonexistent directory")
	}
}

func TestStore_ExperiencePreserved(t *testing.T) {
	buf := NewReplayBuffer(100)
	exp := Experience{
		Context:     "click Submit button",
		Action:      "click",
		ArgsJSON:    `{"x":100,"y":200}`,
		Success:     true,
		CoordX:      100,
		CoordY:      200,
		WindowTitle: "Test Window",
	}
	buf.Store(exp)
	samples := buf.Sample(1)
	if len(samples) != 1 {
		t.Fatal("expected 1 sample")
	}
	s := samples[0]
	if s.Action != "click" || s.CoordX != 100 || s.CoordY != 200 {
		t.Errorf("experience not preserved: %+v", s)
	}
	if s.Context != "click Submit button" {
		t.Errorf("context not preserved: %q", s.Context)
	}
}

func TestReplayBuffer_ConcurrentStore(t *testing.T) {
	buf := NewReplayBuffer(1000)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			buf.Store(Experience{Action: "click"})
			done <- true
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if buf.Size() != 100 {
		t.Errorf("expected 100 after concurrent store, got %d", buf.Size())
	}
}

func TestCapacity_Zero(t *testing.T) {
	buf := NewReplayBuffer(0)
	if buf.Capacity() != 0 {
		t.Errorf("expected capacity 0, got %d", buf.Capacity())
	}
	buf.Store(Experience{Action: "click"})
	if buf.Size() != 0 {
		t.Errorf("expected size 0 for zero-capacity buffer, got %d", buf.Size())
	}
}
