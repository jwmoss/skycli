package cli

import (
	"runtime/debug"
	"testing"
)

func TestResolvedVersionUsesModuleVersionWhenLdflagsMissing(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	}

	v, c, d := resolvedVersion("dev", "", "", info)
	if v != "v1.2.3" {
		t.Fatalf("version: got %q", v)
	}
	if c != "" || d != "" {
		t.Fatalf("commit/date: got %q %q", c, d)
	}
}

func TestResolvedVersionKeepsLdflagVersion(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	}

	v, _, _ := resolvedVersion("v9.9.9", "", "", info)
	if v != "v9.9.9" {
		t.Fatalf("version: got %q", v)
	}
}

func TestResolvedVersionFallsBackToDev(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
	}

	v, _, _ := resolvedVersion("dev", "", "", info)
	if v != "dev" {
		t.Fatalf("version: got %q", v)
	}
}

func TestResolvedVersionUsesBuildVCSMetadata(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "a4f58bf7882ec62f990d562b4b3b9dd1f4c02948"},
			{Key: "vcs.time", Value: "2026-05-18T19:24:58Z"},
		},
	}

	v, c, d := resolvedVersion("dev", "", "", info)
	if v != "dev" {
		t.Fatalf("version: got %q", v)
	}
	if c != "a4f58bf" {
		t.Fatalf("commit: got %q", c)
	}
	if d != "2026-05-18T19:24:58Z" {
		t.Fatalf("date: got %q", d)
	}
}
