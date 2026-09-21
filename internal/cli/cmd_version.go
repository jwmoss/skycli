package cli

import (
	"runtime/debug"
	"strings"
)

func runVersion(rc *runCtx, args []string) int {
	if len(args) > 0 {
		return usage(rc, "skycli version takes no arguments")
	}
	v, c, d := currentVersion()
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]string{
			"version": v,
			"commit":  c,
			"date":    d,
		})
		return exitOK
	}
	rc.out.Line("skycli %s", v)
	if c != "" {
		rc.out.Line("commit %s", c)
	}
	if d != "" {
		rc.out.Line("built %s", d)
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
