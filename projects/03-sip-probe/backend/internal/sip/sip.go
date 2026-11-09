package sip

import (
	"errors"
	"strconv"
	"strings"

	"github.com/scc-platform/shared/fn"
)

var ErrNoStatusLine = errors.New("sip: status line not found")

type Response struct {
	Code   int
	Reason string
	CSeq   string
	Server string
}

func BuildOptions(trunk Trunk, callID string, cseq int) []byte {
	lines := []string{
		"OPTIONS sip:" + trunk.Host + " SIP/2.0",
		"Via: SIP/2.0/" + strings.ToUpper(trunk.Transport) + " " + trunk.LocalAddr + ";branch=z9hG4bK" + callID,
		"Max-Forwards: 70",
		"From: <sip:probe@" + trunk.Host + ">;tag=" + callID,
		"To: <sip:" + trunk.Host + ">",
		"Call-ID: " + callID + "@scc-probe",
		"CSeq: " + strconv.Itoa(cseq) + " OPTIONS",
		"Contact: <sip:probe@" + trunk.LocalAddr + ">",
		"User-Agent: SCC-SIP-Probe/1.0",
		"Accept: application/sdp",
		"Content-Length: 0",
		"",
		"",
	}
	return []byte(strings.Join(lines, "\r\n"))
}

func ParseResponse(raw []byte) (Response, error) {
	return statusOf(strings.Split(string(raw), "\r\n"))
}

func statusOf(lines []string) (Response, error) {
	parsed := parseStatus(firstLine(lines)).withHeaders(lines)
	return parsed, statusErr(parsed.Code)
}

func statusErr(code int) error {
	return fn.Choose(code >= 100, nil, ErrNoStatusLine)
}

func firstLine(lines []string) string {
	return fn.Or(lines[:min(1, len(lines))], "")
}

func parseStatus(line string) Response {
	parts := strings.Split(line, " ")
	return Response{Code: parseCode(parts), Reason: reasonOf(parts)}
}

func parseCode(parts []string) int {
	parsed, err := strconv.Atoi(second(parts))
	return fn.Choose(err == nil, parsed, 0)
}

func second(parts []string) string {
	return fn.Choose(len(parts) > 1, parts[min(1, len(parts))], "")
}

func reasonOf(parts []string) string {
	return strings.Join(parts[min(2, len(parts)):], " ")
}

func (r Response) withHeaders(lines []string) Response {
	return foldHeaders(r, lines)
}

func foldHeaders(start Response, lines []string) Response {
	return foldInto(start, lines)
}

func foldInto(enriched Response, lines []string) Response {
	for _, line := range lines {
		enriched = enriched.withHeader(line)
	}
	return enriched
}

func (r Response) withHeader(line string) Response {
	name, value := splitHeader(line)
	return r.withNamed(strings.TrimSpace(name), value)
}

func (r Response) withNamed(name, value string) Response {
	return fn.With(r, func(next *Response) {
		next.CSeq = fn.Choose(strings.EqualFold(name, "cseq"), value, next.CSeq)
		next.Server = fn.Choose(strings.EqualFold(name, "server"), value, next.Server)
	})
}

func splitHeader(line string) (string, string) {
	name, value, _ := strings.Cut(line, ":")
	return name, strings.TrimSpace(value)
}

func (r Response) Healthy() bool {
	return r.Code >= 200 && r.Code < 300
}

func (r Response) Class() string {
	return strconv.Itoa(r.Code/100) + "xx"
}

type Trunk struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Transport string `json:"transport"`
	LocalAddr string `json:"local_addr"`
	Carrier   string `json:"carrier"`
	Channels  int    `json:"channels"`
	Capacity  int    `json:"capacity"`
}

func (t Trunk) Address() string {
	return t.Host + ":" + strconv.Itoa(t.Port)
}

func (t Trunk) Load() float64 {
	return float64(t.Channels) / float64(max(1, t.Capacity)) * 100
}
