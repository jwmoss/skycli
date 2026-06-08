package cli

import (
	"fmt"
	"runtime/debug"
	"strings"
)

func runVersion(rc *runCtx, args []string) int {
	_ = args
	v, c, d := currentVersion()
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]string{
			"version": v,
			"commit":  c,
			"date":    d,
		})
		return exitOK
	}
	fmt.Fprintf(rc.stdout, "skycli %s\n", v)
	if c != "" {
		fmt.Fprintf(rc.stdout, "commit %s\n", c)
	}
	if d != "" {
		fmt.Fprintf(rc.stdout, "built %s\n", d)
	}
	return exitOK
}

func currentVersion() (string, string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return resolvedVersion(version, commit, date, nil)
	}
	return resolvedVersion(version, commit, date, info)
}

func resolvedVersion(v, c, d string, info *debug.BuildInfo) (string, string, string) {
	if info != nil {
		if isDevelopmentVersion(v) && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if c == "" {
					c = shortCommit(setting.Value)
				}
			case "vcs.time":
				if d == "" {
					d = setting.Value
				}
			}
		}
	}
	if isDevelopmentVersion(v) {
		v = "dev"
	}
	return v, c, d
}

func isDevelopmentVersion(v string) bool {
	return v == "" || v == "dev" || v == "(devel)"
}

func shortCommit(rev string) string {
	rev = strings.TrimSpace(rev)
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}
