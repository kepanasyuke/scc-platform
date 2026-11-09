package cdr

import (
	"context"
	"github.com/scc-platform/shared/fn"
	"sort"
	"sync"
	"time"

	"github.com/scc-platform/shared/lists"
)

type Stats struct {
	Received   int     `json:"received"`
	Accepted   int     `json:"accepted"`
	Rejected   int     `json:"rejected"`
	Duplicates int     `json:"duplicates"`
	AnswerRate float64 `json:"answer_rate"`
	ASASeconds float64 `json:"asa_seconds"`
	SLAPercent float64 `json:"sla_percent"`
	Batches    int     `json:"batches"`
}

type IngestResult struct {
	Accepted   int      `json:"accepted"`
	Rejected   int      `json:"rejected"`
	Duplicates int      `json:"duplicates"`
	Errors     []string `json:"errors"`
}

type Service struct {
	mutex    sync.RWMutex
	sla      time.Duration
	records  map[string]Record
	order    []string
	counters counters
}

type counters struct {
	received   int
	rejected   int
	duplicates int
	batches    int
}

func NewService(sla time.Duration) *Service {
	return &Service{sla: sla, records: make(map[string]Record, 128), order: make([]string, 0, 128)}
}

func (s *Service) SLA() time.Duration {
	return s.sla
}

func (s *Service) Ingest(ctx context.Context, batch []Record) IngestResult {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.ingestLocked(batch)
}

func (s *Service) ingestLocked(batch []Record) IngestResult {
	s.counters = s.counters.opened(len(batch))
	return s.absorbed(batch)
}

func (c counters) opened(size int) counters {
	return fn.With(c, func(next *counters) {
		next.received += size
		next.batches++
	})
}

func (s *Service) absorbed(batch []Record) IngestResult {
	return s.finished(foldInto(tally{}, batch, s.absorb))
}

func foldInto(start tally, batch []Record, absorb func(tally, Record) tally) tally {
	for _, record := range batch {
		start = absorb(start, record)
	}
	return start
}

func (s *Service) finished(tally tally) IngestResult {
	s.counters = s.counters.absorbed(tally)
	return tally.result()
}

func (c counters) absorbed(tally tally) counters {
	return fn.With(c, func(next *counters) {
		next.rejected += tally.rejected
		next.duplicates += tally.duplicates
	})
}

type tally struct {
	accepted   int
	rejected   int
	duplicates int
	problems   []string
}

func (t tally) result() IngestResult {
	return IngestResult{Accepted: t.accepted, Rejected: t.rejected, Duplicates: t.duplicates, Errors: t.problems}
}

func (s *Service) absorb(current tally, record Record) tally {
	return current.add(s.settle(record), record.CallID)
}

func (s *Service) settle(record Record) verdict {
	return s.judge(record, s.classify(record))
}

func (s *Service) judge(record Record, judged verdict) verdict {
	s.accept(record, judged)
	return judged
}

func (t tally) add(verdict verdict, callID string) tally {
	return fn.With(t, func(next *tally) {
		next.accepted += boolToInt(verdict == verdictAccepted)
		next.rejected += boolToInt(verdict == verdictRejected)
		next.duplicates += boolToInt(verdict == verdictDuplicate)
		next.problems = appendProblem(t.problems, verdict, callID)
	})
}

func (s *Service) accept(record Record, verdict verdict) {
	if verdict == verdictAccepted {
		s.records[record.CallID] = record.Normalized(s.sla)
		s.order = append(s.order, record.CallID)
	}
}

type verdict int

const (
	verdictAccepted verdict = iota
	verdictRejected
	verdictDuplicate
)

func (s *Service) classify(record Record) verdict {
	invalid := record.Validate() != nil
	return fn.Choose(invalid, verdictRejected, fn.Choose(s.known(record.CallID), verdictDuplicate, verdictAccepted))
}

func (s *Service) known(callID string) bool {
	_, exists := s.records[callID]
	return exists
}

func appendProblem(problems []string, verdict verdict, callID string) []string {
	return fn.Choose(verdict == verdictRejected, append(problems, callID+": validation failed"), problems)
}

func boolToInt(flag bool) int {
	return int(boolAsNumber(flag))
}

func boolAsNumber(flag bool) int32 {
	return fn.Choose(flag, 1, int32(0))
}

func (s *Service) Recent(limit int) []Record {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.recentLocked(limit)
}

func (s *Service) recentLocked(limit int) []Record {
	return byStartDescending(lists.Map(tailOf(s.order, limit), s.recordOf))
}

func tailOf(order []string, limit int) []string {
	return order[len(order)-min(limit, len(order)):]
}

func byStartDescending(records []Record) []Record {
	sort.SliceStable(records, func(i, j int) bool { return records[i].StartedAt.After(records[j].StartedAt) })
	return records
}

func (s *Service) recordOf(callID string) Record {
	return s.records[callID]
}

func (s *Service) Stats() Stats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.statsLocked()
}

func (s *Service) statsLocked() Stats {
	return s.statsOf(lists.Map(s.order, s.recordOf))
}

func (s *Service) statsOf(records []Record) Stats {
	answered := lists.Filter(records, Record.Answered)
	return Stats{
		Received:   s.counters.received,
		Accepted:   len(s.order),
		Rejected:   s.counters.rejected,
		Duplicates: s.counters.duplicates,
		Batches:    s.counters.batches,
		AnswerRate: ratio(len(answered), len(records)) * 100,
		ASASeconds: average(lists.Map(answered, Record.WaitSeconds)),
		SLAPercent: ratio(len(lists.Filter(records, s.slaMet)), len(records)) * 100,
	}
}

func (s *Service) slaMet(record Record) bool {
	return record.SLAMet
}

func ratio(part, total int) float64 {
	return fn.Choose(total > 0, float64(part)/float64(total), 0.0)
}

func average(values []float64) float64 {
	return safeDivide(lists.Sum(values, identity), len(values))
}

func identity(value float64) float64 {
	return value
}

func safeDivide(total float64, count int) float64 {
	return fn.Choose(count > 0, total/float64(count), 0.0)
}
