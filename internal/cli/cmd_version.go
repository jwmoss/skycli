package cli

import "fmt"

func runVersion(rc *runCtx, args []string) int {
	_ = args
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]string{
			"version": version,
			"commit":  commit,
			"date":    date,
		})
		return exitOK
	}
	fmt.Fprintf(rc.stdout, "skycli %s\n", version)
	if commit != "" {
		fmt.Fprintf(rc.stdout, "commit %s\n", commit)
	}
	if date != "" {
		fmt.Fprintf(rc.stdout, "built %s\n", date)
	}
	return exitOK
}
