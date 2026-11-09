package sip

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/lists"
)

type ProbeResult struct {
	Trunk     string    `json:"trunk"`
	Code      int       `json:"code"`
	Reason    string    `json:"reason"`
	RTTms     float64   `json:"rtt_ms"`
	Healthy   bool      `json:"healthy"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
}

type Prober interface {
	Probe(ctx context.Context, trunk Trunk) ProbeResult
}

type UDPProber struct {
	Timeout time.Duration
}

func (p UDPProber) Probe(ctx context.Context, trunk Trunk) ProbeResult {
	started := time.Now()
	return p.dial(ctx, trunk).measured(started)
}

func (p UDPProber) dial(ctx context.Context, trunk Trunk) ProbeResult {
	conn, err := net.DialTimeout("udp", trunk.Address(), p.Timeout)
	return exchange(ctx, conn, err, trunk)
}

func exchange(ctx context.Context, conn net.Conn, dialErr error, trunk Trunk) ProbeResult {
	base := ProbeResult{Trunk: trunk.Name, CheckedAt: time.Now().UTC()}
	return fn.Either(dialErr == nil,
		func() ProbeResult { return conversed(ctx, conn, base) },
		func() ProbeResult { return base.failed(dialErr) })
}

func conversed(ctx context.Context, conn net.Conn, base ProbeResult) ProbeResult {
	defer conn.Close() //nolint:errcheck // закрытие сокета не влияет на результат опроса
	return roundTrip(ctx, conn, base)
}

func roundTrip(ctx context.Context, conn net.Conn, base ProbeResult) ProbeResult {
	reply, writeErr := send(conn)
	return base.afterWrite(reply, writeErr)
}

func send(conn net.Conn) ([]byte, error) {
	return written(conn, BuildOptions(Trunk{}, "probe", 42))
}

func written(conn net.Conn, request []byte) ([]byte, error) {
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	return connRoundTrip(conn, request)
}

func connRoundTrip(conn net.Conn, request []byte) ([]byte, error) {
	_, err := conn.Write(request)
	return readReply(conn, err), err
}

func readReply(conn net.Conn, writeErr error) []byte {
	return fn.Lazy(writeErr == nil, func() []byte { return incoming(conn) })
}

func incoming(conn net.Conn) []byte {
	buffer := make([]byte, 2048)
	return buffer[:readInto(conn, buffer)]
}

func readInto(conn net.Conn, buffer []byte) int {
	read, _ := conn.Read(buffer)
	return read
}

func (p ProbeResult) afterWrite(reply []byte, writeErr error) ProbeResult {
	return fn.Either(writeErr == nil,
		func() ProbeResult { return p.parsed(reply) },
		func() ProbeResult { return p.failed(writeErr) })
}

func (p ProbeResult) parsed(raw []byte) ProbeResult {
	response, err := ParseResponse(raw)
	return fn.Either(err == nil,
		func() ProbeResult { return p.afterParse(response) },
		func() ProbeResult { return p.failed(err) })
}

func (p ProbeResult) afterParse(response Response) ProbeResult {
	return fn.With(p, func(next *ProbeResult) {
		next.Code = response.Code
		next.Reason = response.Reason
		next.Healthy = response.Healthy()
	})
}

func (p ProbeResult) failed(err error) ProbeResult {
	return fn.With(p, func(next *ProbeResult) { next.Error = err.Error() })
}

func (p ProbeResult) measured(started time.Time) ProbeResult {
	return fn.With(p, func(next *ProbeResult) {
		next.RTTms = float64(time.Since(started).Microseconds()) / 1000
	})
}

type Service struct {
	mutex  sync.RWMutex
	prober Prober
	trunks []Trunk
	last   map[string]ProbeResult
}

func NewService(prober Prober, trunks []Trunk) *Service {
	return &Service{prober: prober, trunks: trunks, last: make(map[string]ProbeResult, len(trunks))}
}

func (s *Service) Trunks() []Trunk {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.trunks
}

func (s *Service) Probe(ctx context.Context, name string) (ProbeResult, error) {
	trunk, found := lists.Find(s.Trunks(), func(item Trunk) bool { return item.Name == name })
	return s.probeFound(ctx, trunk, found)
}

func (s *Service) probeFound(ctx context.Context, trunk Trunk, found bool) (ProbeResult, error) {
	result := fn.Choose(found, s.probeAndStore(ctx, trunk), ProbeResult{Trunk: trunk.Name})
	return result, unknownTrunk(found)
}

func unknownTrunk(found bool) error {
	return fn.Choose(found, nil, ErrUnknownTrunk)
}

func (s *Service) probeAndStore(ctx context.Context, trunk Trunk) ProbeResult {
	return s.stored(trunk, s.prober.Probe(ctx, trunk))
}

func (s *Service) stored(trunk Trunk, result ProbeResult) ProbeResult {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.last[trunk.Name] = result
	return result
}

func (s *Service) ProbeAll(ctx context.Context) []ProbeResult {
	return lists.Map(s.Trunks(), func(trunk Trunk) ProbeResult { return s.probeAndStore(ctx, trunk) })
}

func (s *Service) Last() []ProbeResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.snapshotLocked()
}

func (s *Service) snapshotLocked() []ProbeResult {
	return lists.Map(s.trunks, func(trunk Trunk) ProbeResult { return s.last[trunk.Name] })
}

type Summary struct {
	Trunks  int `json:"trunks"`
	Healthy int `json:"healthy"`
	Failed  int `json:"failed"`
}

func (s *Service) Summary() Summary {
	return summaryOf(s.Last())
}

func summaryOf(results []ProbeResult) Summary {
	healthy := len(lists.Filter(results, healthyProbe))
	return Summary{Trunks: len(results), Healthy: healthy, Failed: len(results) - healthy}
}

func healthyProbe(result ProbeResult) bool {
	return result.Healthy
}
