package pgdoctor

import (
	"fmt"
	"github.com/scc-platform/shared/fn"
	"math"
	"sort"
	"strings"

	"github.com/scc-platform/shared/lists"
)

type Sample struct {
	QueryID     int64   `json:"queryid"`
	Fingerprint string  `json:"fingerprint"`
	Table       string  `json:"table"`
	Calls       int64   `json:"calls"`
	MeanMs      float64 `json:"mean_ms"`
	TotalMs     float64 `json:"total_ms"`
	Rows        int64   `json:"rows"`
	SeqScans    int64   `json:"seq_scans"`
	IdxScans    int64   `json:"index_scans"`
	SharedBlks  int64   `json:"shared_blks_read"`
	Plan        string  `json:"plan"`
}

func (s Sample) SeqShare() float64 {
	total := s.SeqScans + s.IdxScans
	return ratio(s.SeqScans, total) * 100
}

func (s Sample) RowsPerCall() float64 {
	return ratio(s.Rows, s.Calls)
}

func (s Sample) Cost() float64 {
	return s.TotalMs * math.Log1p(float64(s.Calls))
}

type Recommendation struct {
	Fingerprint string  `json:"fingerprint"`
	Table       string  `json:"table"`
	Severity    string  `json:"severity"`
	Problem     string  `json:"problem"`
	Suggestion  string  `json:"suggestion"`
	Estimated   string  `json:"estimated_gain"`
	Score       float64 `json:"score"`
}

type Deadlock struct {
	At       string `json:"at"`
	Victim   string `json:"victim"`
	WaitsFor string `json:"waits_for"`
	LockType string `json:"lock_type"`
	Relation string `json:"relation"`
	Advice   string `json:"advice"`
}

type Analysis struct {
	Samples         int              `json:"samples"`
	SlowQueries     int              `json:"slow_queries"`
	TotalMeanMs     float64          `json:"total_mean_ms"`
	Worst           []Sample         `json:"worst"`
	Recommendations []Recommendation `json:"recommendations"`
	Deadlocks       []Deadlock       `json:"deadlocks"`
	Health          string           `json:"health"`
}

func Analyze(samples []Sample, deadlocks []Deadlock) Analysis {
	return buildAnalysis(samples, deadlocks, collectRecommendations(samples), worstSamples(samples, 5))
}

func buildAnalysis(samples []Sample, deadlocks []Deadlock, recommendations []Recommendation, worst []Sample) Analysis {
	return Analysis{
		Samples:         len(samples),
		SlowQueries:     len(lists.Filter(samples, func(sample Sample) bool { return sample.MeanMs > 200 })),
		TotalMeanMs:     sumMean(samples),
		Worst:           worst,
		Recommendations: recommendations,
		Deadlocks:       deadlocks,
		Health:          health(recommendations),
	}
}

func sumMean(samples []Sample) float64 {
	return lists.Sum(samples, func(sample Sample) float64 { return sample.MeanMs })
}

func worstSamples(samples []Sample, limit int) []Sample {
	return orderedWorst(append(make([]Sample, 0, len(samples)), samples...), limit)
}

func orderedWorst(ordered []Sample, limit int) []Sample {
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Cost() > ordered[j].Cost() })
	return ordered[:min(limit, len(ordered))]
}

func health(recommendations []Recommendation) string {
	critical := lists.Filter(recommendations, func(item Recommendation) bool { return item.Severity == "critical" })
	return healthOf(len(critical) > 0, len(recommendations) > 0)
}

func healthOf(hasCritical, hasAny bool) string {
	return fn.Choose(hasCritical, "critical", fn.Choose(hasAny, "warning", "ok"))
}

func collectRecommendations(samples []Sample) []Recommendation {
	return ranked(lists.Filter(lists.Map(samples, recommend), func(item Recommendation) bool { return item.Problem != "" }))
}

func ranked(items []Recommendation) []Recommendation {
	sort.SliceStable(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	return items
}

func recommend(sample Sample) Recommendation {
	return chooseAdvice(sample, seqScanCase(sample), slowQueryCase(sample), fatScanCase(sample))
}

func chooseAdvice(sample Sample, seqScan, slowQuery, fatScan bool) Recommendation {
	slowOrFat := fn.Choose(slowQuery, slowQueryAdvice(sample), fn.Choose(fatScan, fatScanAdvice(sample), Recommendation{}))
	return fn.Choose(seqScan, seqScanAdvice(sample), slowOrFat)
}

func seqScanCase(sample Sample) bool {
	return sample.SeqShare() > 70 && sample.Calls > 10
}

func slowQueryCase(sample Sample) bool {
	return sample.MeanMs > 800
}

func fatScanCase(sample Sample) bool {
	return sample.RowsPerCall() > 500
}

func seqScanAdvice(sample Sample) Recommendation {
	return Recommendation{
		Fingerprint: sample.Fingerprint,
		Table:       sample.Table,
		Severity:    severity(sample.Calls > 100),
		Problem:     fmt.Sprintf("seq scan %d%% при %d вызовах", int(sample.SeqShare()), sample.Calls),
		Suggestion:  indexSuggestion(sample),
		Estimated:   fmt.Sprintf("−%d%% времени", estimateGain(sample.SeqShare())),
		Score:       sample.Cost(),
	}
}

func indexSuggestion(sample Sample) string {
	target := lastColumn(sample.Plan)
	return fmt.Sprintf("CREATE INDEX CONCURRENTLY idx_%s_%s ON %s (%s) WHERE ...", sample.Table, sanitize(target), sample.Table, target)
}

func lastColumn(plan string) string {
	columns := strings.Split(plan, "::")
	return columns[len(columns)-1]
}

func sanitize(value string) string {
	return strings.ToLower(cleanTarget(strings.TrimSpace(value)))
}

func cleanTarget(clean string) string {
	return strings.NewReplacer(" ", "_", ",", "_", "(", "", ")", "").Replace(clean)
}

func slowQueryAdvice(sample Sample) Recommendation {
	return Recommendation{
		Fingerprint: sample.Fingerprint,
		Table:       sample.Table,
		Severity:    "critical",
		Problem:     fmt.Sprintf("среднее %.0f ms на вызове", sample.MeanMs),
		Suggestion:  "переписать запрос: убрать SELECT *, добавить covering index, проверить plan на Seq Scan по партициям",
		Estimated:   "до −60% времени",
		Score:       sample.Cost(),
	}
}

func fatScanAdvice(sample Sample) Recommendation {
	return Recommendation{
		Fingerprint: sample.Fingerprint,
		Table:       sample.Table,
		Severity:    "warning",
		Problem:     fmt.Sprintf("%.0f строк на вызов", sample.RowsPerCall()),
		Suggestion:  "добавить LIMIT/pagination или partial index по фильтру периода",
		Estimated:   "−40% shared buffers",
		Score:       sample.Cost(),
	}
}

func severity(high bool) string {
	return fn.Choose(high, "critical", "warning")
}

func estimateGain(share float64) int {
	return int(math.Min(80, share*0.8))
}

func ratio(part, total int64) float64 {
	return fn.Choose(total > 0, float64(part)/float64(total), 0.0)
}

func ParseDeadlockLog(entry string) Deadlock {
	lines := strings.Split(entry, "\n")
	return absorbLines(startDeadlock(lines, entry), lines)
}

func startDeadlock(lines []string, entry string) Deadlock {
	return Deadlock{At: firstLine(lines), Advice: adviceFor(entry)}
}

func absorbLines(parsed Deadlock, lines []string) Deadlock {
	for _, line := range lines {
		parsed = parsed.absorb(line)
	}
	return parsed
}

func firstLine(lines []string) string {
	return fn.Choose(len(lines) > 0, strings.TrimSpace(lines[0]), "")
}

func (d Deadlock) absorb(line string) Deadlock {
	return fn.With(d, func(next *Deadlock) {
		next.Victim = pick(next.Victim, line, "process", "is the deadlock victim")
		next.WaitsFor = pick(next.WaitsFor, line, "waits for", "on transaction")
		next.LockType = pick(next.LockType, line, "waits for", "lock on")
		next.Relation = pick(next.Relation, line, "relation", "of relation")
	})
}

func pick(current, line, anchor, tail string) string {
	_, after, found := strings.Cut(line, anchor)
	return fn.Choose(found, trimTail(after, tail), current)
}

func trimTail(value, tail string) string {
	head, _, _ := strings.Cut(value, tail)
	return strings.TrimSpace(head)
}

type adviceRule struct {
	match  func(string) bool
	advice string
}

var deadlockAdvice = []adviceRule{
	{match: shareLocked, advice: "упорядочить захват блокировок: сначала agents, затем queues"},
	{match: func(entry string) bool { return strings.Contains(entry, "tuple") }, advice: "сократить транзакцию и вынести вычисления до UPDATE"},
	{match: func(entry string) bool { return strings.Contains(entry, "relation") }, advice: "использовать CONCURRENTLY для DDL вне окна нагрузки"},
}

func shareLocked(entry string) bool {
	return strings.Contains(entry, "ShareLock") && strings.Contains(entry, "transaction")
}

func adviceFor(entry string) string {
	return adviceOf(deadlockAdvice, entry)
}

func adviceOf(rules []adviceRule, entry string) string {
	matched, found := lists.Find(rules, func(rule adviceRule) bool { return rule.match(entry) })
	return fn.Choose(found, matched.advice, "повторить транзакцию с экспоненциальной задержкой")
}
