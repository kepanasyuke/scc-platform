package esl

import (
	"errors"
	"fmt"
	"github.com/scc-platform/shared/fn"
	"sync"
	"time"

	"github.com/scc-platform/shared/lists"
)

var ErrChannelNotFound = errors.New("esl: channel not found")

type MemorySwitch struct {
	mutex    sync.RWMutex
	channels map[string]Channel
	order    []string
	seq      int
	started  time.Time
}

func NewMemorySwitch() *MemorySwitch {
	return seeded(&MemorySwitch{channels: make(map[string]Channel, 32), started: time.Now()})
}

func seeded(sim *MemorySwitch) *MemorySwitch {
	sim.seed()
	return sim
}

func (m *MemorySwitch) seed() {
	seeds := []Channel{
		{UUID: "b1f2a3c4-0001", Name: "sofia/external/79001234567", Number: "79001234567", Destination: "1001", State: "cs_execute", Direction: "inbound", Codec: "PCMU", MOS: 4.31},
		{UUID: "b1f2a3c4-0002", Name: "sofia/internal/104", Number: "104", Destination: "queue_admission", State: "answered", Direction: "outbound", Codec: "OPUS", MOS: 4.52},
		{UUID: "b1f2a3c4-0003", Name: "sofia/external/79161234567", Number: "79161234567", Destination: "queue_support", State: "cs_ring", Direction: "inbound", Codec: "PCMU", MOS: 3.88},
		{UUID: "b1f2a3c4-0004", Name: "loopback/121", Number: "121", Destination: "conference_42", State: "answered", Direction: "outbound", Codec: "PCMA", MOS: 4.12},
	}
	for _, channel := range seeds {
		m.seq++
		channel.CreatedAt = float64(m.seq * 1000)
		m.channels[channel.UUID] = channel
		m.order = append(m.order, channel.UUID)
	}
}

func (m *MemorySwitch) Channels() []Channel {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return lists.Map(m.order, func(uuid string) Channel { return m.channels[uuid] })
}

func (m *MemorySwitch) Hangup(uuid string) (Channel, error) {
	channel, found := m.channels[uuid]
	return m.closed(uuid, channel.Hangup()), missing(found)
}

func (m *MemorySwitch) closed(uuid string, channel Channel) Channel {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.channels[uuid] = channel
	return channel
}

func missing(found bool) error {
	return fn.Choose(found, nil, ErrChannelNotFound)
}

func (m *MemorySwitch) Originate(destination, extension string) (Channel, error) {
	return m.registered(newChannel(m.nextID(), destination, extension)), nil
}

func (m *MemorySwitch) nextID() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.seq++
	return m.seq
}

func (m *MemorySwitch) registered(channel Channel) Channel {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.channels[channel.UUID] = channel
	return m.ordered(channel)
}

func (m *MemorySwitch) ordered(channel Channel) Channel {
	m.order = append(m.order, channel.UUID)
	return channel
}

func newChannel(seq int, destination, extension string) Channel {
	return Channel{
		UUID:        fmt.Sprintf("gen-%04d", seq),
		Name:        "sofia/internal/" + extension,
		Number:      extension,
		Destination: destination,
		State:       "cs_ring",
		Direction:   "outbound",
		CreatedAt:   float64(time.Now().Unix()),
		Codec:       "PCMU",
		MOS:         4.0,
	}
}

func (m *MemorySwitch) Status() Status {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return Status{
		Connected:   true,
		Host:        "127.0.0.1",
		Port:        8021,
		Sessions:    len(m.order),
		Uptime:      time.Since(m.started).Seconds(),
		Version:     "FreeSWITCH 1.10.12",
		LastCommand: "api status",
	}
}

func Summarize(channels []Channel) Summary {
	return summaryOf(channels, lists.Filter(channels, Channel.Active), lists.Filter(channels, Channel.Talking))
}

func summaryOf(all, active, talking []Channel) Summary {
	return Summary{
		Total:   len(all),
		Active:  len(active),
		Talking: len(talking),
		AvgMOS:  meanMOS(talking),
		Codecs:  codecShare(active),
	}
}

type Summary struct {
	Total   int            `json:"total"`
	Active  int            `json:"active"`
	Talking int            `json:"talking"`
	AvgMOS  float64        `json:"avg_mos"`
	Codecs  map[string]int `json:"codecs"`
}

func meanMOS(channels []Channel) float64 {
	total := lists.Sum(channels, func(channel Channel) float64 { return channel.MOS })
	return fn.Choose(len(channels) > 0, total/float64(len(channels)), 0.0)
}

func codecShare(channels []Channel) map[string]int {
	return shareInto(make(map[string]int, 4), channels)
}

func shareInto(share map[string]int, channels []Channel) map[string]int {
	for _, channel := range channels {
		share[channel.Codec]++
	}
	return share
}
