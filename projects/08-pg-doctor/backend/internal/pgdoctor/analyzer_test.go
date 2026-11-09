package pgdoctor_test

import (
	"testing"

	"github.com/scc-platform/pg-doctor/internal/pgdoctor"
)

func TestSeqScanProducesIndexAdvice(t *testing.T) {
	t.Parallel()
	samples := []pgdoctor.Sample{
		{QueryID: 1, Fingerprint: "select * from cdr", Table: "cdr", Calls: 500, MeanMs: 120, Rows: 400000, SeqScans: 490, IdxScans: 10, Plan: "started_at::queue"},
	}
	analysis := pgdoctor.Analyze(samples, nil)
	assertEqual(t, "рекомендаций", len(analysis.Recommendations), 1)
	assertEqual(t, "здоровье", analysis.Health, "critical")
}

func TestParseDeadlockLogExtractsVictim(t *testing.T) {
	t.Parallel()
	deadlock := pgdoctor.ParseDeadlockLog(pgdoctor.DemoLog())
	assertContains(t, deadlock.Victim, "18422")
	assertContains(t, deadlock.Advice, "блокировок")
}

func TestHealthySampleHasNoRecommendations(t *testing.T) {
	t.Parallel()
	samples := []pgdoctor.Sample{
		{QueryID: 2, Fingerprint: "update agents", Table: "agents", Calls: 9000, MeanMs: 1.2, Rows: 9000, SeqScans: 0, IdxScans: 9000, Plan: "ext"},
	}
	analysis := pgdoctor.Analyze(samples, nil)
	assertEqual(t, "рекомендаций", len(analysis.Recommendations), 0)
	assertEqual(t, "здоровье", analysis.Health, "ok")
}

func assertEqual[T comparable](t *testing.T, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s = %v, ожидалось %v", field, actual, expected)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if haystack[index:index+len(needle)] == needle {
			return
		}
	}
	t.Fatalf("не найдено %q в %q", needle, haystack)
}
