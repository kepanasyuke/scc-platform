package acd_test

import (
	"context"
	"testing"
	"time"

	"github.com/scc-platform/queue-acd/internal/acd"
	"github.com/scc-platform/shared/bus"
)

func TestHighPriorityTicketGoesFirst(t *testing.T) {
	t.Parallel()
	service := newService()
	assertNoError(t, enqueue(service, "caller-low", 0))
	assertNoError(t, enqueue(service, "caller-vip", 9))
	assignment, err := service.Dispatch(context.Background(), "admission")
	assertNoError(t, err)
	assertEqual(t, "caller", assignment.Ticket.Caller, "caller-vip")
}

func TestDispatchWithoutAgentsFails(t *testing.T) {
	t.Parallel()
	service := newService()
	assertNoError(t, enqueue(service, "caller-1", 0))
	service.SetAgentState("101", "busy")
	service.SetAgentState("104", "busy")
	_, err := service.Dispatch(context.Background(), "admission")
	assertEqual(t, "ошибка", err, acd.ErrNoAgents)
}

func TestEnqueueToUnknownQueueFails(t *testing.T) {
	t.Parallel()
	service := newService()
	_, err := service.Enqueue(context.Background(), "sales", "caller-1", "general", 0)
	assertEqual(t, "ошибка", err, acd.ErrUnknownQueue)
}

func TestSnapshotCountsWaiting(t *testing.T) {
	t.Parallel()
	service := newService()
	assertNoError(t, enqueue(service, "caller-1", 0))
	assertNoError(t, enqueue(service, "caller-2", 0))
	assertEqual(t, "в очереди", service.Snapshot("admission").Waiting, 2)
}

func newService() *acd.Service {
	return acd.NewService(bus.NewMemory(), 20*time.Second, acd.DefaultAgents(), acd.DefaultQueues())
}

func enqueue(service *acd.Service, caller string, priority int) error {
	_, err := service.Enqueue(context.Background(), "admission", caller, "general", priority)
	return err
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s = %v, ожидалось %v", field, actual, expected)
	}
}
