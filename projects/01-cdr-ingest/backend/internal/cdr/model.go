package cdr

import (
	"errors"
	"time"

	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/lists"
)

var (
	ErrEmptyCallID  = errors.New("cdr: call_id is required")
	ErrBadDirection = errors.New("cdr: direction must be inbound or outbound")
	ErrBadTimings   = errors.New("cdr: ended_at must be after started_at")
	ErrNegative     = errors.New("cdr: durations must not be negative")
)

type Record struct {
	CallID      string    `json:"call_id"`
	Direction   string    `json:"direction"`
	Trunk       string    `json:"trunk"`
	Queue       string    `json:"queue"`
	AgentExt    string    `json:"agent_ext"`
	StartedAt   time.Time `json:"started_at"`
	AnsweredAt  time.Time `json:"answered_at"`
	EndedAt     time.Time `json:"ended_at"`
	RingSeconds float64   `json:"ring_seconds"`
	TalkSeconds float64   `json:"talk_seconds"`
	HangupCause string    `json:"hangup_cause"`
	SLAMet      bool      `json:"sla_met"`
}

func (r Record) Validate() error {
	return firstFault(r)
}

func firstFault(r Record) error {
	return fn.Or(lists.Filter(r.faults(), notNil), nil)
}

func notNil(err error) bool {
	return err != nil
}

func (r Record) faults() []error {
	return []error{
		fn.Choose(r.CallID == "", ErrEmptyCallID, nil),
		fn.Choose(r.Direction != "inbound" && r.Direction != "outbound", ErrBadDirection, nil),
		fn.Choose(r.EndedAt.Before(r.StartedAt), ErrBadTimings, nil),
		fn.Choose(r.RingSeconds < 0 || r.TalkSeconds < 0, ErrNegative, nil),
	}
}

func (r Record) Answered() bool {
	return !r.AnsweredAt.IsZero() && r.AnsweredAt.After(r.StartedAt)
}

func (r Record) WaitSeconds() float64 {
	return max(0, r.RingSeconds)
}

func (r Record) HandleSeconds() float64 {
	return r.WaitSeconds() + max(0, r.TalkSeconds)
}

func (r Record) SLA(target time.Duration) bool {
	return r.Answered() && r.WaitSeconds() <= target.Seconds()
}

func (r Record) Normalized(target time.Duration) Record {
	return fn.With(r, func(next *Record) {
		next.SLAMet = r.SLA(target)
		next.Queue = fallback(r.Queue, "default")
		next.Trunk = fallback(r.Trunk, "unknown")
		next.HangupCause = fallback(r.HangupCause, "NORMAL_CLEARING")
	})
}

func fallback(value, def string) string {
	return fn.Choose(value == "", def, value)
}
