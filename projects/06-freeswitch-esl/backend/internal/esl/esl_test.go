package esl_test

import (
	"testing"

	"github.com/scc-platform/freeswitch-esl/internal/esl"
)

const authOK = "Content-Type: command/reply\nReply-Text: +OK accepted\n\n"

func TestParseFrameReadsReply(t *testing.T) {
	t.Parallel()
	frame, err := esl.ParseFrame([]byte(authOK))
	assertNoError(t, err)
	assertTrue(t, "ответ OK", frame.OK())
	assertEqual(t, "тип", frame.Header["Content-Type"], "command/reply")
}

func TestParseFrameRejectsTruncated(t *testing.T) {
	t.Parallel()
	_, err := esl.ParseFrame([]byte("Content-Type: command/reply"))
	assertTrue(t, "ошибка разбора", err != nil)
}

func TestHangupMarksChannelClosed(t *testing.T) {
	t.Parallel()
	sw := esl.NewMemorySwitch()
	first := sw.Channels()[0]
	closed, err := sw.Hangup(first.UUID)
	assertNoError(t, err)
	assertTrue(t, "канал закрыт", !closed.Active())
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func assertTrue(t *testing.T, message string, flag bool) {
	t.Helper()
	if !flag {
		t.Fatalf("условие не выполнено: %s", message)
	}
}

func assertEqual[T comparable](t *testing.T, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s = %v, ожидалось %v", field, actual, expected)
	}
}
