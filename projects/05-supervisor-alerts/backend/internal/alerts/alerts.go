package alerts

import (
	"fmt"
	"github.com/scc-platform/shared/fn"
	"sort"
	"sync"
	"time"

	"github.com/scc-platform/shared/bus"
	"github.com/scc-platform/shared/lists"
)

type Metrics struct {
	Queue          string  `json:"queue"`
	Waiting        int     `json:"waiting"`
	OldestWaitSec  float64 `json:"oldest_wait_sec"`
	AgentsReady    int     `json:"agents_ready"`
	AgentsBusy     int     `json:"agents_busy"`
	AbandonmentPct float64 `json:"abandonment_pct"`
	SLAPercent     float64 `json:"sla_percent"`
	TrunkFailed    bool    `json:"trunk_failed"`
	ActiveChannels int     `json:"active_channels"`
}

type Rule struct {
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Threshold  float64 `json:"threshold"`
	Severity   string  `json:"severity"`
	SilenceSec int     `json:"silence_sec"`
}

var ruleChecks = map[string]func(Rule, Metrics) bool{
	"waiting":         func(rule Rule, metrics Metrics) bool { return float64(metrics.Waiting) > rule.Threshold },
	"oldest_wait":     func(rule Rule, metrics Metrics) bool { return metrics.OldestWaitSec > rule.Threshold },
	"sla":             func(rule Rule, metrics Metrics) bool { return metrics.SLAPercent < rule.Threshold },
	"abandonment":     func(rule Rule, metrics Metrics) bool { return metrics.AbandonmentPct > rule.Threshold },
	"trunk":           func(rule Rule, metrics Metrics) bool { return metrics.TrunkFailed },
	"no_ready_agents": func(rule Rule, metrics Metrics) bool { return metrics.AgentsReady == 0 && rule.Threshold > 0 },
	"channels":        func(rule Rule, metrics Metrics) bool { return float64(metrics.ActiveChannels) > rule.Threshold },
}

func (r Rule) Broken(metrics Metrics) bool {
	check := ruleChecks[r.Kind]
	return check != nil && check(r, metrics)
}

type Alert struct {
	ID             string            `json:"id"`
	Rule           string            `json:"rule"`
	Severity       string            `json:"severity"`
	Subject        string            `json:"subject"`
	Message        string            `json:"message"`
	Values         map[string]string `json:"values"`
	CreatedAt      time.Time         `json:"created_at"`
	AcknowledgedAt time.Time         `json:"acknowledged_at,omitzero"`
	Silenced       bool              `json:"silenced"`
}

func (a Alert) Acknowledged() bool {
	return !a.AcknowledgedAt.IsZero()
}

func (a Alert) Acknowledge(now time.Time) Alert {
	return fn.With(a, func(next *Alert) { next.AcknowledgedAt = now })
}

type Service struct {
	mutex  sync.RWMutex
	hub    bus.Bus
	rules  []Rule
	alerts []Alert
	silent map[string]time.Time
	seq    int
	now    func() time.Time
}

func NewService(hub bus.Bus, rules []Rule) *Service {
	return &Service{hub: hub, rules: rules, alerts: make([]Alert, 0, 64), silent: make(map[string]time.Time, 16), now: time.Now}
}

func (s *Service) Rules() []Rule {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.rules
}

func (s *Service) Evaluate(metrics []Metrics) []Alert {
	return s.publish(s.evaluateAll(metrics))
}

func (s *Service) evaluateAll(metrics []Metrics) []Alert {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.collectAlerts(metrics)
}

func (s *Service) collectAlerts(metrics []Metrics) (raised []Alert) {
	for _, item := range metrics {
		raised = append(raised, s.evaluateQueue(item)...)
	}
	return
}

func (s *Service) evaluateQueue(metrics Metrics) []Alert {
	return lists.Filter(lists.Map(s.rules, func(rule Rule) Alert { return s.raise(rule, metrics) }), raisedAlert)
}

func raisedAlert(alert Alert) bool {
	return alert.ID != ""
}

func (s *Service) raise(rule Rule, metrics Metrics) Alert {
	return *s.record(rule, metrics, s.compose(rule, metrics))
}

func (s *Service) record(rule Rule, metrics Metrics, alert *Alert) *Alert {
	fn.Do(rule.Broken(metrics) && !s.silenced(rule.Name, metrics.Queue), func() { s.stamp(rule, metrics, alert) })
	return alert
}

func (s *Service) stamp(rule Rule, metrics Metrics, alert *Alert) {
	alert.ID = s.nextID()
	s.store(rule, metrics, *alert)
}

func (s *Service) nextID() string {
	s.seq++
	return fmt.Sprintf("a-%04d", s.seq)
}

func (s *Service) store(rule Rule, metrics Metrics, alert Alert) {
	s.alerts = append(s.alerts, alert)
	s.silent[keyOf(rule, metrics)] = s.now().Add(time.Duration(rule.SilenceSec) * time.Second)
}

func keyOf(rule Rule, metrics Metrics) string {
	return rule.Name + "|" + metrics.Queue
}

func (s *Service) silenced(rule, queue string) bool {
	until, exists := s.silent[rule+"|"+queue]
	return exists && until.After(s.now())
}

func (s *Service) compose(rule Rule, metrics Metrics) *Alert {
	return &Alert{
		Rule:      rule.Name,
		Severity:  rule.Severity,
		Subject:   metrics.Queue,
		Message:   message(rule, metrics),
		Values:    values(metrics),
		CreatedAt: s.now().UTC(),
	}
}

func message(rule Rule, metrics Metrics) string {
	return fmt.Sprintf("правило %s сработало на очереди %s (порог %.1f)", rule.Name, metrics.Queue, rule.Threshold)
}

func values(metrics Metrics) map[string]string {
	return map[string]string{
		"waiting":      fmt.Sprintf("%d", metrics.Waiting),
		"oldest_wait":  fmt.Sprintf("%.1f", metrics.OldestWaitSec),
		"sla":          fmt.Sprintf("%.1f", metrics.SLAPercent),
		"abandonment":  fmt.Sprintf("%.1f", metrics.AbandonmentPct),
		"agents_ready": fmt.Sprintf("%d", metrics.AgentsReady),
	}
}

func (s *Service) publish(fresh []Alert) []Alert {
	for _, alert := range fresh {
		if alert.ID != "" {
			_ = s.hub.Publish("alerts.raised", []byte(alert.Rule))
		}
	}
	return fresh
}

func (s *Service) Alerts(open bool) []Alert {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.filterLocked(open)
}

func (s *Service) filterLocked(open bool) []Alert {
	kept := lists.Filter(s.alerts, func(alert Alert) bool { return alert.Acknowledged() != open })
	return sortedRecent(append(make([]Alert, 0, len(kept)), kept...))
}

func sortedRecent(alerts []Alert) []Alert {
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].CreatedAt.After(alerts[j].CreatedAt) })
	return alerts
}

func (s *Service) Acknowledge(id string) (Alert, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	alert, found := lists.Find(s.alerts, func(item Alert) bool { return item.ID == id })
	return alert, s.replaceAcknowledged(alert, found)
}

func (s *Service) replaceAcknowledged(alert Alert, found bool) error {
	s.alerts = s.replace(s.alerts, alert.Acknowledge(s.now().UTC()), found)
	return acknowledgeErr(found)
}

func (s *Service) replace(alerts []Alert, updated Alert, ok bool) []Alert {
	return foldReplaced(make([]Alert, 0, len(alerts)), alerts, updated, ok)
}

func foldReplaced(next []Alert, alerts []Alert, updated Alert, ok bool) []Alert {
	for _, alert := range alerts {
		next = appendReplaced(next, alert, updated, alert.ID == updated.ID, ok)
	}
	return next
}

func appendReplaced(next []Alert, alert, updated Alert, same bool, ok bool) []Alert {
	return append(next, fn.Choose(same && ok, updated, alert))
}

func acknowledgeErr(found bool) error {
	return fn.Choose(found, nil, ErrUnknownAlert)
}

func (s *Service) Stats() Stats {
	return statsOf(s.Alerts(false), len(s.rules))
}

func statsOf(alerts []Alert, rules int) Stats {
	critical := lists.Filter(alerts, func(alert Alert) bool { return alert.Severity == "critical" })
	return Stats{Open: len(alerts), Critical: len(critical), Rules: rules}
}

type Stats struct {
	Open     int `json:"open"`
	Critical int `json:"critical"`
	Rules    int `json:"rules"`
}
