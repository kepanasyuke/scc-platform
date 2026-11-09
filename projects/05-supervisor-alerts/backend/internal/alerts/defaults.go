package alerts

import "errors"

var ErrUnknownAlert = errors.New("alerts: unknown alert")

func DefaultRules() []Rule {
	return []Rule{
		{Name: "очередь растёт", Kind: "waiting", Threshold: 15, Severity: "warning", SilenceSec: 300},
		{Name: "долгое ожидание", Kind: "oldest_wait", Threshold: 90, Severity: "critical", SilenceSec: 120},
		{Name: "SLA ниже нормы", Kind: "sla", Threshold: 90, Severity: "warning", SilenceSec: 600},
		{Name: "много отказов", Kind: "abandonment", Threshold: 12, Severity: "critical", SilenceSec: 600},
		{Name: "транк недоступен", Kind: "trunk", Threshold: 1, Severity: "critical", SilenceSec: 60},
		{Name: "нет свободных агентов", Kind: "no_ready_agents", Threshold: 1, Severity: "warning", SilenceSec: 180},
		{Name: "перегрузка каналов", Kind: "channels", Threshold: 180, Severity: "warning", SilenceSec: 300},
	}
}

func DemoMetrics() []Metrics {
	return []Metrics{
		{Queue: "admission", Waiting: 21, OldestWaitSec: 128, AgentsReady: 0, AgentsBusy: 4, AbandonmentPct: 7.4, SLAPercent: 88.2, ActiveChannels: 64},
		{Queue: "support", Waiting: 9, OldestWaitSec: 41, AgentsReady: 2, AgentsBusy: 3, AbandonmentPct: 4.1, SLAPercent: 96.4, ActiveChannels: 51},
		{Queue: "exams", Waiting: 3, OldestWaitSec: 12, AgentsReady: 1, AgentsBusy: 1, AbandonmentPct: 1.8, SLAPercent: 99.1, ActiveChannels: 22},
		{Queue: "callback", Waiting: 0, OldestWaitSec: 0, AgentsReady: 2, AgentsBusy: 0, AbandonmentPct: 0, SLAPercent: 100, ActiveChannels: 9},
		{Queue: "outbound", Waiting: 2, OldestWaitSec: 35, AgentsReady: 3, AgentsBusy: 2, AbandonmentPct: 2.2, SLAPercent: 97.5, TrunkFailed: true, ActiveChannels: 197},
	}
}
