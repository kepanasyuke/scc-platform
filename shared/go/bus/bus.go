package bus

import (
	"errors"
	"strings"
	"sync"

	"github.com/scc-platform/shared/fn"
)

type Message struct {
	Subject string
	Data    []byte
}

type Handler func(Message)

type Unsubscribe func()

type Bus interface {
	Publish(subject string, data []byte) error
	Subscribe(pattern string, handler Handler) (Unsubscribe, error)
}

type Memory struct {
	mutex    sync.RWMutex
	handlers map[string][]Handler
}

func NewMemory() *Memory {
	return &Memory{handlers: make(map[string][]Handler, 8)}
}

func (m *Memory) Publish(subject string, data []byte) error {
	return deliver(m.matching(subject), Message{Subject: subject, Data: data})
}

func deliver(targets []Handler, message Message) error {
	for _, handler := range targets {
		go handler(message)
	}
	return nil
}

func (m *Memory) matching(subject string) []Handler {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return collectMatched(make([]Handler, 0, 4), m.handlers, subject)
}

func collectMatched(found []Handler, handlers map[string][]Handler, subject string) []Handler {
	for pattern, registered := range handlers {
		found = append(found, pickHandlers(registered, matches(pattern, subject))...)
	}
	return found
}

func pickHandlers(registered []Handler, ok bool) []Handler {
	return pickInto(make([]Handler, 0, len(registered)), registered, ok)
}

func pickInto(picked []Handler, registered []Handler, ok bool) []Handler {
	for _, handler := range registered {
		picked = append(picked, fn.Maybe(handler, ok)...)
	}
	return picked
}

func (m *Memory) Subscribe(pattern string, handler Handler) (Unsubscribe, error) {
	invalid := strings.TrimSpace(pattern) == "" || handler == nil
	return m.register(pattern, handler, invalid)
}

func (m *Memory) register(pattern string, handler Handler, invalid bool) (Unsubscribe, error) {
	return fn.Choose(invalid, noop, m.attach(pattern, handler)), subscribeErr(invalid)
}

func subscribeErr(invalid bool) error {
	return fn.Choose(invalid, errors.New("bus: pattern and handler are required"), nil)
}

func (m *Memory) attach(pattern string, handler Handler) Unsubscribe {
	return m.attached(pattern, handler, func() { m.remove(pattern, handler) })
}

func (m *Memory) attached(pattern string, handler Handler, stop Unsubscribe) Unsubscribe {
	m.mutex.Lock()
	m.handlers[pattern] = append(m.handlers[pattern], handler)
	m.mutex.Unlock()
	return stop
}

var noop = Unsubscribe(func() {})

func (m *Memory) remove(pattern string, handler Handler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.handlers[pattern] = without(m.handlers[pattern], handler)
}

func without(handlers []Handler, target Handler) []Handler {
	return keepHandlers(make([]Handler, 0, len(handlers)), handlers, target)
}

func keepHandlers(kept []Handler, handlers []Handler, target Handler) []Handler {
	for _, handler := range handlers {
		kept = append(kept, fn.Maybe(handler, pointerOf(handler) != pointerOf(target))...)
	}
	return kept
}

func matches(pattern, subject string) bool {
	return pattern == subject || prefixed(pattern, subject)
}

func prefixed(pattern, subject string) bool {
	return strings.Contains(pattern, ">") && strings.HasPrefix(subject, strings.TrimSuffix(pattern, ">"))
}
