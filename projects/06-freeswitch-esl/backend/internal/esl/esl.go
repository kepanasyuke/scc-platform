package esl

import (
	"errors"
	"github.com/scc-platform/shared/fn"
	"strconv"
	"strings"
)

var ErrMalformedFrame = errors.New("esl: malformed frame")

type Frame struct {
	Header map[string]string
	Body   string
}

func (f Frame) Reply() string {
	return strings.TrimSpace(f.Header["Reply-Text"])
}

func (f Frame) OK() bool {
	return strings.HasPrefix(f.Reply(), "+OK")
}

func (f Frame) Value(key string) string {
	return f.Header[key]
}

func ParseFrame(raw []byte) (Frame, error) {
	return splitFrame(string(raw))
}

func splitFrame(text string) (Frame, error) {
	head, body, found := strings.Cut(text, "\n\n")
	return frameOf(head, body, found)
}

func frameOf(head, body string, found bool) (Frame, error) {
	return Frame{Header: parseHeaders(head), Body: body}, frameErr(found)
}

func frameErr(found bool) error {
	return fn.Choose(found, nil, ErrMalformedFrame)
}

func parseHeaders(head string) map[string]string {
	return headersInto(make(map[string]string, 8), head)
}

func headersInto(headers map[string]string, head string) map[string]string {
	for _, line := range strings.Split(head, "\n") {
		name, value, ok := strings.Cut(line, ": ")
		if ok {
			headers[name] = value
		}
	}
	return headers
}

func ContentLength(raw []byte) int {
	frame, err := ParseFrame(raw)
	return lengthOf(frame, err)
}

func lengthOf(frame Frame, err error) int {
	parsed, convErr := strconv.Atoi(frame.Header["Content-Length"])
	return fn.Choose(err == nil && convErr == nil, parsed, 0)
}

func EncodeCommand(command string) []byte {
	return []byte(command + "\n\n")
}

func EncodeAuth(password string) []byte {
	return []byte("auth " + password + "\n\n")
}

type Channel struct {
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	Number      string  `json:"number"`
	Destination string  `json:"destination"`
	State       string  `json:"state"`
	Direction   string  `json:"direction"`
	CreatedAt   float64 `json:"created_at"`
	AnsweredAt  float64 `json:"answered_at"`
	Codec       string  `json:"codec"`
	MOS         float64 `json:"mos"`
}

func (c Channel) Active() bool {
	return c.State != "hangup" && c.State != "cs_destroy"
}

func (c Channel) Talking() bool {
	return c.State == "cs_execute" || c.State == "answered"
}

func (c Channel) Hangup() Channel {
	return fn.With(c, func(next *Channel) { next.State = "hangup" })
}

type Switch interface {
	Channels() []Channel
	Hangup(uuid string) (Channel, error)
	Originate(destination, extension string) (Channel, error)
	Status() Status
}

type Status struct {
	Connected   bool    `json:"connected"`
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	Sessions    int     `json:"sessions"`
	Uptime      float64 `json:"uptime_sec"`
	Version     string  `json:"version"`
	LastCommand string  `json:"last_command"`
}
