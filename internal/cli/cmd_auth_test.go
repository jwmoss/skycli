package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type testFDReader struct {
	fd uintptr
}

func (r testFDReader) Read(_ []byte) (int, error) {
	return 0, errors.New("unexpected fallback read")
}

func (r testFDReader) Fd() uintptr {
	return r.fd
}

func TestReadLoginPasswordUsesHiddenTerminalInput(t *testing.T) {
	origIsTerminal := stdinIsTerminal
	origReadSecret := readTerminalSecret
	t.Cleanup(func() {
		stdinIsTerminal = origIsTerminal
		readTerminalSecret = origReadSecret
	})

	var isTerminalFD, readFD int
	stdinIsTerminal = func(fd int) bool {
		isTerminalFD = fd
		return true
	}
	readTerminalSecret = func(fd int) ([]byte, error) {
		readFD = fd
		return []byte("secret-pw\n"), nil
	}

	var stderr bytes.Buffer
	got, err := readLoginPassword(&runCtx{
		stdin:  testFDReader{fd: 42},
		stderr: &stderr,
	}, false)
	if err != nil {
		t.Fatalf("readLoginPassword returned error: %v", err)
	}
	if got != "secret-pw" {
		t.Fatalf("password: got %q", got)
	}
	if isTerminalFD != 42 || readFD != 42 {
		t.Fatalf("fd: isTerminal=%d read=%d", isTerminalFD, readFD)
	}
	if strings.Contains(stderr.String(), "secret-pw") {
		t.Fatalf("stderr exposed password: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Paste password") {
		t.Fatalf("terminal path used fallback prompt: %q", stderr.String())
	}
}

func TestReadLoginPasswordFallsBackToLineReader(t *testing.T) {
	origIsTerminal := stdinIsTerminal
	origReadSecret := readTerminalSecret
	t.Cleanup(func() {
		stdinIsTerminal = origIsTerminal
		readTerminalSecret = origReadSecret
	})

	stdinIsTerminal = func(_ int) bool {
		return false
	}
	readTerminalSecret = func(_ int) ([]byte, error) {
		t.Fatal("readTerminalSecret should not be called")
		return nil, nil
	}

	var stderr bytes.Buffer
	got, err := readLoginPassword(&runCtx{
		stdin:  strings.NewReader("secret-pw\n"),
		stderr: &stderr,
	}, false)
	if err != nil {
		t.Fatalf("readLoginPassword returned error: %v", err)
	}
	if got != "secret-pw" {
		t.Fatalf("password: got %q", got)
	}
	if !strings.Contains(stderr.String(), "Paste password") {
		t.Fatalf("stderr missing fallback prompt: %q", stderr.String())
	}
}
