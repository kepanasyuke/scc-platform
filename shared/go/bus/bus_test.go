package bus_test

import (
	"sync"
	"testing"
	"time"

	"github.com/scc-platform/shared/bus"
)

func TestPublishDeliversToWildcardSubscriber(t *testing.T) {
	t.Parallel()
	hub := bus.NewMemory()
	received := make(chan bus.Message, 1)
	_, err := hub.Subscribe("calls.>", func(message bus.Message) { received <- message })
	publishErr := hub.Publish("calls.queue.enqueue", []byte(`{"id":1}`))
	assertNoError(t, err)
	assertNoError(t, publishErr)
	select {
	case message := <-received:
		assertSubject(t, message.Subject, "calls.queue.enqueue")
	case <-time.After(time.Second):
		t.Fatal("сообщение не доставлено")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	t.Parallel()
	hub := bus.NewMemory()
	var mutex sync.Mutex
	counter := 0
	stop, err := hub.Subscribe("alerts.>", func(message bus.Message) {
		mutex.Lock()
		counter++
		mutex.Unlock()
	})
	assertNoError(t, err)
	stop()
	assertNoError(t, hub.Publish("alerts.sla", nil))
	time.Sleep(50 * time.Millisecond)
	assertCount(t, counter, 0)
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func assertSubject(t *testing.T, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("subject = %q, ожидался %q", actual, expected)
	}
}

func assertCount(t *testing.T, actual, expected int) {
	t.Helper()
	if actual != expected {
		t.Fatalf("получено %d сообщений, ожидалось %d", actual, expected)
	}
}
