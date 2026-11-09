package cdr_test

import (
	"context"
	"testing"
	"time"

	"github.com/scc-platform/cdr-ingest/internal/cdr"
)

func TestIngestDeduplicatesByCallID(t *testing.T) {
	t.Parallel()
	service := cdr.NewService(20 * time.Second)
	result := service.Ingest(context.Background(), []cdr.Record{sample("call-1"), sample("call-1")})
	assertEqual(t, "accepted", result.Accepted, 1)
	assertEqual(t, "duplicates", result.Duplicates, 1)
}

func TestIngestRejectsBrokenRecords(t *testing.T) {
	t.Parallel()
	service := cdr.NewService(20 * time.Second)
	broken := sample("call-2")
	broken.Direction = "sideways"
	result := service.Ingest(context.Background(), []cdr.Record{broken})
	assertEqual(t, "rejected", result.Rejected, 1)
	assertEqual(t, "accepted", result.Accepted, 0)
}

func TestSLAIsMetWhenAnsweredFast(t *testing.T) {
	t.Parallel()
	service := cdr.NewService(20 * time.Second)
	service.Ingest(context.Background(), []cdr.Record{sample("call-3")})
	assertEqual(t, "sla percent", int(service.Stats().SLAPercent), 100)
}

func sample(callID string) cdr.Record {
	started := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	return cdr.Record{
		CallID:      callID,
		Direction:   "inbound",
		Trunk:       "mti-sip-01",
		Queue:       "admission",
		AgentExt:    "101",
		StartedAt:   started,
		AnsweredAt:  started.Add(8 * time.Second),
		EndedAt:     started.Add(3 * time.Minute),
		RingSeconds: 8,
		TalkSeconds: 172,
	}
}

func assertEqual[T comparable](t *testing.T, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s = %v, ожидалось %v", field, actual, expected)
	}
}
