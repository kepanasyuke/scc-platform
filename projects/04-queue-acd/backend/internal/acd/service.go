package acd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/scc-platform/shared/bus"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/lists"
)

type Service struct {
	mutex   sync.RWMutex
	bus     bus.Bus
	sla     time.Duration
	queues  map[string][]string
	agents  map[string]Agent
	tickets map[string][]Ticket
	closed  map[string]bool
	seq     int
}

func NewService(hub bus.Bus, sla time.Duration, agents []Agent, queues map[string][]string) *Service {
	return &Service{
		bus:     hub,
		sla:     sla,
		agents:  indexAgents(agents),
		queues:  queues,
		tickets: make(map[string][]Ticket, len(queues)),
		closed:  make(map[string]bool, len(queues)),
	}
}

func indexAgents(agents []Agent) map[string]Agent {
	return indexInto(make(map[string]Agent, len(agents)), agents)
}

func indexInto(index map[string]Agent, agents []Agent) map[string]Agent {
	for _, agent := range agents {
		index[agent.Ext] = agent
	}
	return index
}

func (s *Service) Agents() []Agent {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return lists.Map(s.memberExts(), s.agentOf)
}

func (s *Service) memberExts() []string {
	return lists.Keys(s.agents)
}

func (s *Service) agentOf(ext string) Agent {
	return s.agents[ext]
}

func (s *Service) Queues() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return lists.Keys(s.queues)
}

func (s *Service) SetAgentState(ext, state string) (Agent, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.switchState(ext, state)
}

func (s *Service) switchState(ext, state string) (Agent, error) {
	agent, found := s.agents[ext]
	return s.saved(ext, agent.switchTo(state)), knownAgent(found)
}

func (s *Service) saved(ext string, agent Agent) Agent {
	s.agents[ext] = agent
	return agent
}

func knownAgent(found bool) error {
	return fn.Choose(found, nil, ErrUnknownAgent)
}

func (s *Service) Enqueue(ctx context.Context, queue, caller, skill string, priority int) (Ticket, error) {
	ticket, err := s.collect(queue, caller, skill, priority)
	return s.accepted(ctx, ticket, err)
}

func (s *Service) collect(queue, caller, skill string, priority int) (Ticket, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.enqueueLocked(queue, caller, skill, priority)
}

func (s *Service) accepted(ctx context.Context, ticket Ticket, err error) (Ticket, error) {
	fn.Do(err == nil, func() { s.publish(ctx, "calls.queue.enqueue", ticket) })
	return ticket, err
}

func (s *Service) publish(ctx context.Context, subject string, payload any) {
	data, _ := json.Marshal(payload)
	_ = s.bus.Publish(subject, data)
}

func (s *Service) enqueueLocked(queue, caller, skill string, priority int) (Ticket, error) {
	ticket := s.newTicket(queue, caller, skill, priority)
	return ticket, s.record(queue, ticket)
}

func (s *Service) record(queue string, ticket Ticket) error {
	return s.recorded(queue, ticket, queueErr(s.known(queue), s.closed[queue]))
}

func (s *Service) recorded(queue string, ticket Ticket, err error) error {
	s.tickets[queue] = appendTicket(s.tickets[queue], ticket, err == nil)
	return err
}

func (s *Service) known(queue string) bool {
	_, present := s.queues[queue]
	return present
}

func queueErr(known, closed bool) error {
	return fn.Choose(!known, ErrUnknownQueue, fn.Choose(closed, ErrClosed, nil))
}

func appendTicket(tickets []Ticket, ticket Ticket, ok bool) []Ticket {
	return append(tickets, fn.Maybe(ticket, ok)...)
}

func (s *Service) newTicket(queue, caller, skill string, priority int) Ticket {
	s.seq++
	return Ticket{
		ID:         fmt.Sprintf("t-%05d", s.seq),
		Queue:      queue,
		Caller:     caller,
		Skill:      skill,
		Priority:   priority,
		EnqueuedAt: time.Now().UTC(),
		State:      "waiting",
	}
}

func (s *Service) Dispatch(ctx context.Context, queue string) (Assignment, error) {
	assignment, err := s.take(queue)
	return s.dispatched(ctx, assignment, err)
}

func (s *Service) take(queue string) (Assignment, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.dispatchLocked(queue)
}

func (s *Service) dispatched(ctx context.Context, assignment Assignment, err error) (Assignment, error) {
	fn.Do(err == nil, func() { s.publish(ctx, "calls.queue.assign", assignment) })
	return assignment, err
}

func (s *Service) dispatchLocked(queue string) (Assignment, error) {
	assignment, err := s.assignFrom(queue)
	return s.cleared(queue, assignment, err)
}

func (s *Service) cleared(queue string, assignment Assignment, err error) (Assignment, error) {
	s.tickets[queue] = removeTicket(s.tickets[queue], assignment.Ticket.ID)
	return assignment, err
}

func (s *Service) assignFrom(queue string) (Assignment, error) {
	head, hasTickets := s.headOf(queue)
	return s.assign(queue, head, hasTickets)
}

func (s *Service) headOf(queue string) (Ticket, bool) {
	return lists.Find(sortTickets(s.waitingOf(queue)), func(Ticket) bool { return true })
}

func (s *Service) waitingOf(queue string) []Ticket {
	now := time.Now().UTC()
	return lists.Map(s.tickets[queue], func(ticket Ticket) Ticket { return ticket.Waiting(now) })
}

func removeTicket(tickets []Ticket, id string) []Ticket {
	return keptTickets(make([]Ticket, 0, len(tickets)), tickets, id)
}

func keptTickets(kept []Ticket, tickets []Ticket, id string) []Ticket {
	for _, ticket := range tickets {
		kept = append(kept, fn.Maybe(ticket, ticket.ID != id)...)
	}
	return kept
}

func (s *Service) assign(queue string, head Ticket, hasTickets bool) (Assignment, error) {
	agent, found := s.pickAgent(head)
	return s.committed(agent, assignmentOf(head, agent), assignmentErr(hasTickets, found))
}

func assignmentOf(head Ticket, agent Agent) Assignment {
	return Assignment{Ticket: head, Agent: agent, WaitSec: head.WaitSeconds}
}

func (s *Service) committed(agent Agent, assignment Assignment, err error) (Assignment, error) {
	fn.Do(err == nil, func() { s.agents[agent.Ext] = agent.Busy() })
	return assignment, err
}

func assignmentErr(hasTickets, foundAgent bool) error {
	return fn.Choose(!hasTickets, ErrNoTickets, fn.Choose(!foundAgent, ErrNoAgents, nil))
}

func (s *Service) pickAgent(ticket Ticket) (Agent, bool) {
	return lists.Find(s.sortedCandidates(ticket), func(Agent) bool { return true })
}

func (s *Service) sortedCandidates(ticket Ticket) []Agent {
	candidates := lists.Filter(s.agentsIn(ticket.Queue), func(agent Agent) bool { return agent.Handles(ticket.Skill) })
	return sortAgentsBy(candidates)
}

func (s *Service) agentsIn(queue string) []Agent {
	return lists.Map(s.queues[queue], s.agentOf)
}

func (s *Service) Snapshot(queue string) Snapshot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.snapshotLocked(queue)
}

func (s *Service) snapshotLocked(queue string) Snapshot {
	return snapshotOf(queue, s.waitingOf(queue), s.membersOf(queue), s.sla)
}

func snapshotOf(queue string, waiting []Ticket, members []Agent, sla time.Duration) Snapshot {
	return Snapshot{
		Queue:   queue,
		Waiting: len(waiting),
		Ready:   len(lists.Filter(members, Agent.Ready)),
		Busy:    len(members) - len(lists.Filter(members, Agent.Ready)),
		Oldest:  oldestWait(waiting),
		AvgWait: meanWaits(waiting),
		SLAPct:  slaPercent(waiting, sla),
		Tickets: waiting,
		Members: members,
	}
}

func (s *Service) membersOf(queue string) []Agent {
	return lists.Map(s.queues[queue], s.agentOf)
}

func oldestWait(tickets []Ticket) float64 {
	return peakOf(0, tickets)
}

func peakOf(highest float64, tickets []Ticket) float64 {
	for _, ticket := range tickets {
		highest = max(highest, ticket.WaitSeconds)
	}
	return highest
}

func meanWaits(tickets []Ticket) float64 {
	return averageOf(lists.Map(tickets, func(ticket Ticket) float64 { return ticket.WaitSeconds }))
}

func averageOf(values []float64) float64 {
	return lists.Sum(values, func(value float64) float64 { return value }) / float64(max(1, len(values)))
}

func slaPercent(tickets []Ticket, sla time.Duration) float64 {
	within := lists.Filter(tickets, func(ticket Ticket) bool { return ticket.WaitSeconds <= sla.Seconds() })
	return float64(len(within)) / float64(max(1, len(tickets))) * 100
}
