package acd

import (
	"errors"
	"sort"
	"time"

	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/lists"
)

var (
	ErrUnknownQueue = errors.New("acd: unknown queue")
	ErrNoAgents     = errors.New("acd: no available agents")
	ErrUnknownAgent = errors.New("acd: unknown agent")
	ErrClosed       = errors.New("acd: queue is closed")
	ErrNoTickets    = errors.New("acd: queue has no waiting tickets")
)

type Agent struct {
	Ext        string    `json:"ext"`
	Name       string    `json:"name"`
	Team       string    `json:"team"`
	Skills     []string  `json:"skills"`
	State      string    `json:"state"`
	Handled    int       `json:"handled"`
	LastCallAt time.Time `json:"last_call_at"`
}

func (a Agent) Ready() bool {
	return a.State == "ready"
}

func (a Agent) Handles(skill string) bool {
	return a.HasSkill(skill) && a.Ready()
}

func (a Agent) HasSkill(skill string) bool {
	return lists.Contains(a.Skills, skill)
}

func (a Agent) Busy() Agent {
	return fn.With(a.counted(), func(agent *Agent) { agent.stateBusy() })
}

func (a *Agent) stateBusy() {
	a.State = "busy"
	a.LastCallAt = time.Now().UTC()
}

func (a Agent) counted() Agent {
	return fn.With(a, func(agent *Agent) { agent.Handled++ })
}

func (a Agent) Idle() Agent {
	return fn.With(a, func(agent *Agent) { agent.State = "ready" })
}

func (a Agent) Paused() Agent {
	return fn.With(a, func(agent *Agent) { agent.State = "paused" })
}

func (a Agent) switchTo(state string) Agent {
	return fn.Choose(state == "ready", a.Idle(), fn.Choose(state == "busy", a.Busy(), a.Paused()))
}

type Ticket struct {
	ID          string    `json:"id"`
	Queue       string    `json:"queue"`
	Caller      string    `json:"caller"`
	Skill       string    `json:"skill"`
	Priority    int       `json:"priority"`
	EnqueuedAt  time.Time `json:"enqueued_at"`
	Agent       string    `json:"agent"`
	State       string    `json:"state"`
	WaitSeconds float64   `json:"wait_seconds"`
}

func (t Ticket) Waiting(now time.Time) Ticket {
	return fn.With(t, func(ticket *Ticket) { ticket.WaitSeconds = now.Sub(t.EnqueuedAt).Seconds() })
}

func (t Ticket) Rank() int {
	return -t.Priority
}

type Assignment struct {
	Ticket      Ticket `json:"ticket"`
	Agent       Agent  `json:"agent"`
	WaitSec     float64 `json:"wait_seconds"`
	SLABreached bool   `json:"sla_breached"`
}

type Snapshot struct {
	Queue   string   `json:"queue"`
	Waiting int      `json:"waiting"`
	Ready   int      `json:"ready"`
	Busy    int      `json:"busy"`
	Oldest  float64  `json:"oldest_wait"`
	AvgWait float64  `json:"avg_wait"`
	SLAPct  float64  `json:"sla_percent"`
	Tickets []Ticket `json:"tickets"`
	Members []Agent  `json:"members"`
}

func sortTickets(tickets []Ticket) []Ticket {
	return sortedTickets(make([]Ticket, 0, len(tickets)), tickets)
}

func sortedTickets(out []Ticket, tickets []Ticket) []Ticket {
	return sortedByPriority(append(out, tickets...))
}

func sortedByPriority(tickets []Ticket) []Ticket {
	sort.SliceStable(tickets, func(i, j int) bool { return ahead(tickets[i], tickets[j]) })
	return tickets
}

func ahead(left, right Ticket) bool {
	return left.Priority > right.Priority || (left.Priority == right.Priority && left.EnqueuedAt.Before(right.EnqueuedAt))
}
