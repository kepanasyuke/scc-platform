package sip_test

import (
	"testing"

	"github.com/scc-platform/sip-probe/internal/sip"
)

const okReply = "SIP/2.0 200 OK\r\nCSeq: 42 OPTIONS\r\nServer: FreeSWITCH\r\nContent-Length: 0\r\n\r\n"

func TestParseResponseReadsStatus(t *testing.T) {
	t.Parallel()
	reply, err := sip.ParseResponse([]byte(okReply))
	assertNoError(t, err)
	assertEqual(t, "код", reply.Code, 200)
	assertEqual(t, "класс", reply.Class(), "2xx")
	assertTrue(t, "здоров", reply.Healthy())
}

func TestParseResponseRejectsGarbage(t *testing.T) {
	t.Parallel()
	_, err := sip.ParseResponse([]byte("not a sip reply"))
	assertTrue(t, "ошибка разбора", err != nil)
}

func TestBuildOptionsContainsMandatoryHeaders(t *testing.T) {
	t.Parallel()
	trunk := sip.Trunk{Name: "t", Host: "10.0.0.1", Port: 5060, Transport: "udp", LocalAddr: "10.0.0.2:5060"}
	request := string(sip.BuildOptions(trunk, "abc", 7))
	assertContains(t, request, "OPTIONS sip:10.0.0.1 SIP/2.0")
	assertContains(t, request, "CSeq: 7 OPTIONS")
	assertContains(t, request, "Call-ID: abc@scc-probe")
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

func assertTrue(t *testing.T, message string, flag bool) {
	t.Helper()
	if !flag {
		t.Fatalf("условие не выполнено: %s", message)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	found := false
	for index := 0; index+len(needle) <= len(haystack); index++ {
		found = found || haystack[index:index+len(needle)] == needle
	}
	if !found {
		t.Fatalf("не найдено %q", needle)
	}
}
