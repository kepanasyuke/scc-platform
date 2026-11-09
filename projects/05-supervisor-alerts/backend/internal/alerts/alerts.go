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

func (r Rule) Broken(metrics Metrics) bool {
	switch r.Kind {
	case "waiting":
		return float64(metrics.Waiting) > r.Threshold
	case "oldest_wait":
		return metrics.OldestWaitSec > r.Threshold
	case "sla":
		return metrics.SLAPercent < r.Threshold
	case "abandonment":
		return metrics.AbandonmentPct > r.Threshold
	case "trunk":
		return metrics.TrunkFailed
	case "no_ready_agents":
		return metrics.AgentsReady == 0 && r.Threshold > 0
	case "channels":
		return float64(metrics.ActiveChannels) > r.Threshold
	}
	return false
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
	next := a
	next.AcknowledgedAt = now
	return next
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
	s.mutex.Lock()
	fresh := make([]Alert, 0, len(metrics))
	for _, item := range metrics {
		fresh = append(fresh, s.evaluateQueue(item)...)
	}
	s.mutex.Unlock()
	s.publish(fresh)
	return fresh
}

func (s *Service) evaluateQueue(metrics Metrics) []Alert {
	raised := make([]Alert, 0, len(s.rules))
	for _, rule := range s.rules {
		raised = appendAlert(raised, s.raise(rule, metrics))
	}
	return raised
}

func appendAlert(raised []Alert, alert Alert) []Alert {
	return fn.Choose(alert.ID != "", append(raised, alert), raised)
}

func (s *Service) raise(rule Rule, metrics Metrics) Alert {
	alert := s.compose(rule, metrics)
	silenced := s.silenced(rule.Name, metrics.Queue)
	if rule.Broken(metrics) && !silenced {
		s.seq++
		alert.ID = fmt.Sprintf("a-%04d", s.seq)
		s.alerts = append(s.alerts, alert)
		s.silent[rule.Name+"|"+metrics.Queue] = s.now().Add(time.Duration(rule.SilenceSec) * time.Second)
	}
	return alert
}

func (s *Service) silenced(rule, queue string) bool {
	until, exists := s.silent[rule+"|"+queue]
	return exists && until.After(s.now())
}

func (s *Service) compose(rule Rule, metrics Metrics) Alert {
	return Alert{
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

func (s *Service) publish(fresh []Alert) {
	for _, alert := range fresh {
		if alert.ID != "" {
			_ = s.hub.Publish("alerts.raised", []byte(alert.Rule))
		}
	}
}

func (s *Service) Alerts(open bool) []Alert {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.filterLocked(open)
}

func (s *Service) filterLocked(open bool) []Alert {
	kept := lists.Filter(s.alerts, func(alert Alert) bool { return alert.Acknowledged() != open })
	sorted := append(make([]Alert, 0, len(kept)), kept...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].CreatedAt.After(sorted[j].CreatedAt) })
	return sorted
}

func (s *Service) Acknowledge(id string) (Alert, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	alert, found := lists.Find(s.alerts, func(item Alert) bool { return item.ID == id })
	s.alerts = s.replace(s.alerts, alert.Acknowledge(s.now().UTC()), found)
	return alert, acknowledgeErr(found)
}

func (s *Service) replace(alerts []Alert, updated Alert, ok bool) []Alert {
	next := make([]Alert, 0, len(alerts))
	for _, alert := range alerts {
		next = appendReplaced(next, alert, updated, alert.ID == updated.ID, ok)
	}
	return next
}

func appendReplaced(next []Alert, alert, updated Alert, same bool, ok bool) []Alert {
	chosen := alert
	if same && ok {
		chosen = updated
	}
	return append(next, chosen)
}

func acknowledgeErr(found bool) error {
	return fn.Choose(found, nil, ErrUnknownAlert)
}

func (s *Service) Stats() Stats {
	alerts := s.Alerts(false)
	critical := lists.Filter(alerts, func(alert Alert) bool { return alert.Severity == "critical" })
	return Stats{Open: len(alerts), Critical: len(critical), Rules: len(s.rules)}
}

type Stats struct {
	Open     int `json:"open"`
	Critical int `json:"critical"`
	Rules    int `json:"rules"`
}
