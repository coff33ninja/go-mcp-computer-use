package dataloader

import "context"

// Sample represents a single training example from the datalog.
type Sample struct {
	Context     string  // OCR text snapshot
	Action      string  // tool name (click, hover, type_text, etc.)
	ArgsJSON    string  // tool arguments as JSON string
	Success     bool    // whether the action succeeded
	CoordX      int     // predicted X coordinate (0 if N/A)
	CoordY      int     // predicted Y coordinate (0 if N/A)
	WindowTitle string  // window title at time of action
	DPI         float64 // screen DPI scale at time of action
	CreatedAt   string  // timestamp of the sample
}

// Sequence represents a chain of consecutive actions from the same session.
type Sequence struct {
	Context  string   // OCR text before the first action
	Actions  []Action // ordered actions in the sequence
}

// Action is a single step in a Sequence.
type Action struct {
	Action  string // tool name
	ArgsJSON string // tool arguments as JSON
}

// Loader reads training samples from the SQLite datalog database.
type Loader interface {
	// LoadAll returns all available training samples.
	LoadAll(ctx context.Context) ([]Sample, error)

	// LoadRecent returns the N most recent training samples.
	LoadRecent(ctx context.Context, n int) ([]Sample, error)

	// LoadByTool returns training samples filtered by tool name.
	LoadByTool(ctx context.Context, tool string) ([]Sample, error)

	// LoadSequences returns consecutive action sequences grouped by session_id.
	// minLen is the minimum number of actions in a sequence to include.
	LoadSequences(ctx context.Context, minLen int) ([]Sequence, error)

	// Count returns the total number of training samples.
	Count(ctx context.Context) (int, error)

	// Stats returns per-tool sample counts.
	Stats(ctx context.Context) (map[string]int, error)
}
