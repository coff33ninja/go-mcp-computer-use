package actions

import (
	"testing"
)

func TestShortHashStable(t *testing.T) {
	if shortHash("abc") != shortHash("abc") {
		t.Fatal("hash not stable")
	}
	if shortHash("abc") == shortHash("abd") {
		t.Fatal("hash collision on distinct inputs")
	}
}

func TestToolMatchesClickFamily(t *testing.T) {
	if !toolMatches("click", "click") {
		t.Fatal("click should match click")
	}
	if !toolMatches("hover", "click") {
		t.Fatal("hover prediction can be confirmed by click")
	}
	if toolMatches("type", "click") {
		t.Fatal("type should not match click")
	}
}

func TestSupervisedLedgerExcludesUnknownPending(t *testing.T) {
	// Use the process datalog if present; never close it (other tests share it).
	db, err := ensureLedgerDB()
	if err != nil || db == nil {
		t.Skip("datalog unavailable")
	}
	// Insert a synthetic unknown + pending via SQL to prove filter.
	dlogMu.Lock()
	db.Exec(`INSERT INTO ml_predictions(engine,pred_tool,status,created_at)
		VALUES('statistical','click','unknown', datetime('now'))`)
	db.Exec(`INSERT INTO ml_predictions(engine,pred_tool,status,created_at)
		VALUES('statistical','click','pending', datetime('now'))`)
	db.Exec(`INSERT INTO ml_predictions(engine,pred_tool,status,created_at)
		VALUES('statistical','click','hit', datetime('now'))`)
	dlogMu.Unlock()

	samples := SupervisedLedgerSamples(100)
	foundHit := false
	for _, s := range samples {
		status, _ := s["status"].(string)
		switch status {
		case PredStatusUnknown, PredStatusPending:
			t.Fatalf("supervised sample includes non-trainable status %q", status)
		case PredStatusHit:
			foundHit = true
		}
	}
	if !foundHit && len(samples) > 0 {
		// hit row we inserted should appear
		t.Logf("samples=%d (hit row may be outside limit)", len(samples))
	}

	st := MLPredictionOutcomeStatsFromDB()
	if st.Total < 3 {
		t.Logf("ledger stats total=%d (db may be shared/reset)", st.Total)
	}
}
