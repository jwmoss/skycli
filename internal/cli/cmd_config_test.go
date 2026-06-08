package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSplitEditorCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"vi", []string{"vi"}},
		{"code -w", []string{"code", "-w"}},
		{"vim -n", []string{"vim", "-n"}},
		{`"/path/with space/editor" --flag`, []string{"/path/with space/editor", "--flag"}},
		{"  emacs   -nw  ", []string{"emacs", "-nw"}},
	}
	for _, tc := range cases {
		got, err := splitEditorCommand(tc.in)
		if err != nil {
			t.Fatalf("splitEditorCommand(%q): %v", tc.in, err)
		}
		if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
			t.Fatalf("splitEditorCommand(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
	if _, err := splitEditorCommand(`code "-w`); err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

// config edit must invoke an EDITOR that carries its own arguments, passing both
// the configured argument and the config path through to the editor process.
func TestConfigEditPassesEditorArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a POSIX shell helper; argv parsing is covered by TestSplitEditorCommand")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	script := filepath.Join(dir, "fake-editor.sh")
	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	t.Setenv("EDITOR", script+" --flag")
	t.Setenv("VISUAL", "")

	cfgPath := filepath.Join(dir, "config.json")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "config", "edit"},
		strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstderr=%s", code, stderr.String())
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	lines := strings.Fields(string(got))
	if len(lines) != 2 || lines[0] != "--flag" || lines[1] != cfgPath {
		t.Fatalf("editor received %q, want [--flag %s]", string(got), cfgPath)
	}
}
