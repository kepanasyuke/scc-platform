package analytics

import (
	"sort"
	"time"

	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/lists"
)

type Call struct {
	Queue       string    `json:"queue"`
	Agent       string    `json:"agent"`
	StartedAt   time.Time `json:"started_at"`
	WaitSeconds float64   `json:"wait_seconds"`
	TalkSeconds float64   `json:"talk_seconds"`
	Answered    bool      `json:"answered"`
}

type Bucket struct {
	Hour     string  `json:"hour"`
	Total    int     `json:"total"`
	Answered int     `json:"answered"`
	AvgWait  float64 `json:"avg_wait"`
	AvgTalk  float64 `json:"avg_talk"`
}

type QueueKPI struct {
	Queue       string  `json:"queue"`
	Total       int     `json:"total"`
	Answered    int     `json:"answered"`
	ASA         float64 `json:"asa"`
	P95Wait     float64 `json:"p95_wait"`
	SLA         float64 `json:"sla"`
	Abandonment float64 `json:"abandonment"`
	Load        float64 `json:"load"`
}

type Summary struct {
	Total     int     `json:"total"`
	Answered  int     `json:"answered"`
	SLA       float64 `json:"sla"`
	ASA       float64 `json:"asa"`
	P95Wait   float64 `json:"p95_wait"`
	PeakHour  string  `json:"peak_hour"`
	PeakCount int     `json:"peak_count"`
}

type Service struct {
	calls []Call
	sla   time.Duration
}

func NewService(calls []Call, sla time.Duration) *Service {
	return &Service{calls: calls, sla: sla}
}

func (s *Service) Hourly() []Bucket {
	groups := lists.GroupBy(s.calls, hourKey)
	return lists.Map(sortedKeys(groups), func(key string) Bucket { return bucketOf(key, groups[key]) })
}

func hourKey(call Call) string {
	return call.StartedAt.UTC().Format("15:00")
}

func sortedKeys(groups map[string][]Call) []string {
	return sortedStrings(make([]string, 0, len(groups)), groups)
}

func sortedStrings(keys []string, groups map[string][]Call) []string {
	return sortedKeysOf(keysInto(keys, groups))
}

func keysInto(keys []string, groups map[string][]Call) []string {
	for key := range groups {
		keys = append(keys, key)
	}
	return keys
}

func sortedKeysOf(keys []string) []string {
	sort.Strings(keys)
	return keys
}

func bucketOf(key string, calls []Call) Bucket {
	answered := lists.Filter(calls, answeredCall)
	return Bucket{
		Hour:     key,
		Total:    len(calls),
		Answered: len(answered),
		AvgWait:  mean(lists.Map(answered, waitOf)),
		AvgTalk:  mean(lists.Map(answered, talkOf)),
	}
}

func answeredCall(call Call) bool {
	return call.Answered
}

func waitOf(call Call) float64 {
	return call.WaitSeconds
}

func talkOf(call Call) float64 {
	return call.TalkSeconds
}

func mean(values []float64) float64 {
	return lists.Sum(values, identity) / float64(max(1, len(values)))
}

func identity(value float64) float64 {
	return value
}

func (s *Service) Queues() []QueueKPI {
	groups := lists.GroupBy(s.calls, queueOf)
	return lists.Map(sortedKeys(groups), func(key string) QueueKPI { return s.kpiOf(key, groups[key]) })
}

func queueOf(call Call) string {
	return call.Queue
}

func (s *Service) kpiOf(queue string, calls []Call) QueueKPI {
	answered := lists.Filter(calls, answeredCall)
	return s.kpiOfAnswered(queue, calls, answered)
}

func (s *Service) kpiOfAnswered(queue string, calls, answered []Call) QueueKPI {
	return reportOf(queue, calls, answered, lists.Map(answered, waitOf), lists.Filter(answered, s.fastEnough))
}

func reportOf(queue string, calls, answered []Call, waits []float64, withinSLA []Call) QueueKPI {
	return QueueKPI{
		Queue:       queue,
		Total:       len(calls),
		Answered:    len(answered),
		ASA:         mean(waits),
		P95Wait:     percentile(waits, 0.95),
		SLA:         percent(len(withinSLA), len(calls)),
		Abandonment: percent(len(calls)-len(answered), len(calls)),
		Load:        mean(lists.Map(answered, talkOf)) * float64(len(answered)),
	}
}

func (s *Service) fastEnough(call Call) bool {
	return call.WaitSeconds <= s.sla.Seconds()
}

func percent(part, total int) float64 {
	return float64(part) / float64(max(1, total)) * 100
}

func percentile(values []float64, share float64) float64 {
	return pickPosition(sortedCopy(values), position(len(values), share))
}

func sortedCopy(values []float64) []float64 {
	return sortedFloats(append(make([]float64, 0, len(values)), values...))
}

func sortedFloats(values []float64) []float64 {
	sort.Float64s(values)
	return values
}

func position(length int, share float64) int {
	return min(max(int(float64(length)*share), 0), max(0, length-1))
}

func pickPosition(sorted []float64, index int) float64 {
	return sorted[min(index, max(0, len(sorted)-1))]
}

func (s *Service) Summary() Summary {
	answered := lists.Filter(s.calls, answeredCall)
	return summaryOf(s, answered, s.peakHour())
}

func summaryOf(s *Service, answered []Call, peak Bucket) Summary {
	return Summary{
		Total:     len(s.calls),
		Answered:  len(answered),
		SLA:       percent(len(lists.Filter(answered, s.fastEnough)), len(s.calls)),
		ASA:       mean(lists.Map(answered, waitOf)),
		P95Wait:   percentile(lists.Map(answered, waitOf), 0.95),
		PeakHour:  peak.Hour,
		PeakCount: peak.Total,
	}
}

func (s *Service) peakHour() Bucket {
	return peakOfBucket(s.Hourly())
}

func peakOfBucket(buckets []Bucket) Bucket {
	return foldPeak(Bucket{}, buckets)
}

func foldPeak(peak Bucket, buckets []Bucket) Bucket {
	for _, bucket := range buckets {
		peak = pickPeak(peak, bucket)
	}
	return peak
}

func pickPeak(current, candidate Bucket) Bucket {
	return fn.Choose(candidate.Total > current.Total, candidate, current)
}
