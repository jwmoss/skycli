package cli

import (
	"fmt"
	"net/http"
	"strings"
)

var readOnlyCommands = map[string]bool{
	"analytics":            true,
	"auth status":          true,
	"bounties list":        true,
	"bounty list":          true,
	"calendar list":        true,
	"calendar sources":     true,
	"calendar week":        true,
	"commands":             true,
	"categories":           true,
	"category":             true,
	"chores list":          true,
	"chores streak":        true,
	"chores week":          true,
	"chore list":           true,
	"chore streak":         true,
	"chore week":           true,
	"config get":           true,
	"config show":          true,
	"doctor":               true,
	"export":               true,
	"frames":               true,
	"frames avatars":       true,
	"frames colors":        true,
	"frames devices":       true,
	"frames list":          true,
	"frames show":          true,
	"frame":                true,
	"frame avatars":        true,
	"frame colors":         true,
	"frame devices":        true,
	"frame list":           true,
	"frame show":           true,
	"grocery list":         true,
	"grocery show":         true,
	"home":                 true,
	"lists all":            true,
	"lists info":           true,
	"lists list":           true,
	"lists show":           true,
	"lists task-box":       true,
	"lists task-box-items": true,
	"list all":             true,
	"list info":            true,
	"list list":            true,
	"list show":            true,
	"list task-box":        true,
	"list task-box-items":  true,
	"meals categories":     true,
	"meals recipe-info":    true,
	"meals recipes":        true,
	"meals sittings":       true,
	"meal categories":      true,
	"meal recipe-info":     true,
	"meal recipes":         true,
	"meal sittings":        true,
	"photos list":          true,
	"photo list":           true,
	"rewards list":         true,
	"rewards points":       true,
	"reward list":          true,
	"reward points":        true,
	"routines list":        true,
	"routine list":         true,
	"status":               true,
	"version":              true,
	"watch":                true,
}

func enforceSafety(g *globals, args []string) error {
	path := commandPath(args)
	if matchesAny(path, splitCommandList(g.deny)) {
		return fmt.Errorf("command %q is denied by --deny-commands", path)
	}
	allow := splitCommandList(g.allow)
	if len(allow) > 0 && !matchesAny(path, allow) {
		return fmt.Errorf("command %q is not allowed by --allow-commands", path)
	}
	if g.readOnly && !isReadOnlyInvocation(args) {
		return fmt.Errorf("readonly mode blocks mutating command %q", path)
	}
	return nil
}

func commandPath(args []string) string {
	parts := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
		if len(parts) == 2 {
			break
		}
	}
	return strings.Join(parts, " ")
}

func isReadOnlyInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "raw" {
		return rawIsGET(args[1:])
	}
	path := commandPath(args)
	if readOnlyCommands[path] {
		return true
	}
	if len(args) == 1 && readOnlyCommands[args[0]] {
		return true
	}
	return false
}

func rawIsGET(args []string) bool {
	method := http.MethodGet
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--method", "-method":
			if i+1 < len(args) {
				method = strings.ToUpper(args[i+1])
				i++
			}
		default:
			if strings.HasPrefix(args[i], "--method=") {
				method = strings.ToUpper(strings.TrimPrefix(args[i], "--method="))
			}
		}
	}
	return method == http.MethodGet
}

func splitCommandList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func matchesAny(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+" ") {
			return true
		}
	}
	return false
}
