package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/config"
	"github.com/jwmoss/skycli/internal/skylight"
)

const (
	exitOK    = 0
	exitErr   = 1
	exitUsage = 2

	refreshSkew = 5 * time.Minute
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

type globals struct {
	configPath string
	asJSON     bool
	plain      bool
	doctor     bool
	timeout    time.Duration
	traceHTTP  bool
	dryRun     bool
	readOnly   bool
	allow      string
	deny       string
	token      string // override
	frame      int64  // override default frame
}

func (g *globals) register(fs *flag.FlagSet) {
	fs.StringVar(&g.configPath, "config", "", "path to config.json (default: $XDG_CONFIG_HOME/skycli/config.json)")
	fs.BoolVar(&g.asJSON, "json", false, "emit JSON to stdout (default: human-readable table/text)")
	fs.BoolVar(&g.plain, "plain", false, "emit stable TSV/plain output where available")
	fs.BoolVar(&g.doctor, "doctor", false, "run readonly token/API connectivity checks and exit")
	fs.DurationVar(&g.timeout, "timeout", 30*time.Second, "HTTP timeout")
	fs.BoolVar(&g.traceHTTP, "trace-http", false, "log every HTTP request to stderr (no secrets)")
	fs.BoolVar(&g.dryRun, "dry-run", false, "refuse all non-GET HTTP calls")
	fs.BoolVar(&g.readOnly, "readonly", false, "block mutating commands and non-GET HTTP calls")
	fs.StringVar(&g.allow, "allow-commands", "", "comma-separated allowed command prefixes")
	fs.StringVar(&g.deny, "deny-commands", "", "comma-separated denied command prefixes")
	fs.StringVar(&g.token, "token", "", "override access token (also: SKYLIGHT_ACCESS_TOKEN)")
	fs.Int64Var(&g.frame, "frame", 0, "override default frame ID (also: SKYLIGHT_FRAME_ID)")
}

type runCtx struct {
	ctx    context.Context
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	g      *globals
	cfg    *config.Config
	out    *printer

	secretsLoaded bool
}

func (rc *runCtx) requireToken() (string, string, error) {
	rc.loadConfiguredSecrets()
	if rc.g.token != "" {
		return rc.g.token, schemeOrDefault(rc.cfg.AuthScheme), nil
	}
	if env := os.Getenv("SKYLIGHT_ACCESS_TOKEN"); env != "" {
		return env, schemeOrDefault(os.Getenv("SKYLIGHT_AUTH_SCHEME")), nil
	}
	if rc.shouldRefreshConfiguredToken() {
		if _, _, err := rc.refreshConfiguredToken(false); err != nil {
			if rc.cfg.AccessToken == "" || tokenExpired(rc.cfg.AccessTokenExpAt) {
				return "", "", fmt.Errorf("refresh access token: %w", err)
			}
			fmt.Fprintf(rc.stderr, "warning: refresh access token: %v\n", err)
		}
	}
	if rc.cfg.AccessToken == "" {
		return "", "", errors.New("no access token configured — run `skycli auth login`, `skycli auth import-mac`, or `skycli auth set-token`")
	}
	return rc.cfg.AccessToken, schemeOrDefault(rc.cfg.AuthScheme), nil
}

func tokenExpired(t time.Time) bool {
	return !t.IsZero() && time.Now().After(t)
}

func (rc *runCtx) shouldRefreshConfiguredToken() bool {
	if rc.cfg.RefreshToken == "" {
		return false
	}
	if rc.cfg.AccessToken == "" {
		return true
	}
	return !rc.cfg.AccessTokenExpAt.IsZero() && time.Now().Add(refreshSkew).After(rc.cfg.AccessTokenExpAt)
}

// refreshConfiguredToken redeems the stored refresh token for a new access
// token. Skylight rotates refresh tokens, so concurrent invocations are
// serialized with a file lock; unless force is set, credentials are re-read
// after acquiring the lock and the network call is skipped when another
// invocation already refreshed.
func (rc *runCtx) refreshConfiguredToken(force bool) (*skylight.OAuthTokenResponse, time.Time, error) {
	if rc.cfg.RefreshToken == "" {
		return nil, time.Time{}, errors.New("no refresh token configured")
	}
	if rc.cfg.DeviceFingerprint == "" {
		return nil, time.Time{}, errors.New("no device fingerprint configured — re-run `skycli auth import-mac`")
	}
	if unlock, err := acquireLockFile(rc.refreshLockPath()); err != nil {
		fmt.Fprintf(rc.stderr, "warning: lock token refresh: %v\n", err)
	} else {
		defer unlock()
	}
	if !force {
		if err := rc.reloadStoredCredentials(); err == nil && !rc.shouldRefreshConfiguredToken() {
			tok := &skylight.OAuthTokenResponse{
				AccessToken:  rc.cfg.AccessToken,
				RefreshToken: rc.cfg.RefreshToken,
			}
			return tok, rc.cfg.AccessTokenExpAt, nil
		}
	}
	opts := []skylight.Option{
		skylight.WithTimeout(rc.g.timeout),
		skylight.WithAPIVersion(rc.cfg.APIVersion),
	}
	if rc.g.traceHTTP {
		opts = append(opts, skylight.WithTrace(func(method, url string, status int, d time.Duration) {
			fmt.Fprintf(rc.stderr, "[http] %s %s -> %d (%s)\n", method, url, status, d)
		}))
	}
	c := skylight.New(rc.cfg.BaseURL, "", opts...)
	tok, err := c.RefreshOAuthToken(rc.ctx, rc.cfg.RefreshToken, rc.cfg.DeviceFingerprint)
	if err != nil {
		return nil, time.Time{}, err
	}

	expiresAt := time.Time{}
	if tok.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	rc.cfg.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		rc.cfg.RefreshToken = tok.RefreshToken
	}
	rc.cfg.AccessTokenExpAt = expiresAt
	rc.cfg.AuthScheme = config.DefaultAuthScheme
	if err := rc.saveConfiguredSecrets(); err != nil {
		return nil, time.Time{}, err
	}
	if err := rc.saveConfig(); err != nil {
		return nil, time.Time{}, err
	}
	if err := rc.loadSecretsIntoConfig(); err != nil {
		return nil, time.Time{}, err
	}
	return tok, expiresAt, nil
}

// refreshLockPath derives the refresh lock file from the active config path so
// invocations sharing a config serialize against the same lock.
func (rc *runCtx) refreshLockPath() string {
	path := rc.g.configPath
	if path == "" {
		if p, err := config.DefaultPath(); err == nil {
			path = p
		} else {
			path = filepath.Join(os.TempDir(), "skycli-config.json")
		}
	}
	return path + ".lock"
}

// reloadStoredCredentials re-reads tokens from disk (config file plus the
// configured secrets backend) into the in-memory config.
func (rc *runCtx) reloadStoredCredentials() error {
	cfg, err := config.Load(rc.g.configPath)
	if err != nil {
		return err
	}
	rc.cfg.AccessToken = cfg.AccessToken
	rc.cfg.RefreshToken = cfg.RefreshToken
	rc.cfg.AccessTokenExpAt = cfg.AccessTokenExpAt
	backend := rc.secretsBackend()
	if backend == secretsBackendConfig {
		return nil
	}
	secrets, err := rc.readSecrets(backend)
	if err != nil {
		return err
	}
	if secrets.AccessToken != "" {
		rc.cfg.AccessToken = secrets.AccessToken
	}
	if secrets.RefreshToken != "" {
		rc.cfg.RefreshToken = secrets.RefreshToken
	}
	return nil
}

func schemeOrDefault(s string) string {
	if s == "" {
		return config.DefaultAuthScheme
	}
	return s
}

func (rc *runCtx) client() (*skylight.Client, error) {
	token, scheme, err := rc.requireToken()
	if err != nil {
		return nil, err
	}
	opts := []skylight.Option{
		skylight.WithTimeout(rc.g.timeout),
		skylight.WithDryRun(rc.g.dryRun),
		skylight.WithReadOnly(rc.g.readOnly),
		skylight.WithAuthScheme(scheme),
		skylight.WithAPIVersion(rc.cfg.APIVersion),
	}
	if rc.g.traceHTTP {
		opts = append(opts, skylight.WithTrace(func(method, url string, status int, d time.Duration) {
			fmt.Fprintf(rc.stderr, "[http] %s %s -> %d (%s)\n", method, url, status, d)
		}))
	}
	return skylight.New(rc.cfg.BaseURL, token, opts...), nil
}

func (rc *runCtx) requireFrame() (int64, error) {
	if rc.g.frame != 0 {
		return rc.g.frame, nil
	}
	if env := os.Getenv("SKYLIGHT_FRAME_ID"); env != "" {
		id, err := skylight.ParseID(env)
		if err == nil && id != 0 {
			return id, nil
		}
	}
	if rc.cfg.DefaultFrameID != 0 {
		return rc.cfg.DefaultFrameID, nil
	}
	return 0, errors.New("no frame ID configured — run `skycli frames` to list frame IDs, then pass --frame <id>, set SKYLIGHT_FRAME_ID, or run `skycli frames set-default <id>`")
}

type command struct {
	name    string
	summary string
	run     func(rc *runCtx, args []string) int
}

func topLevelCommands() []command {
	return []command{
		{"commands", "print the command catalog for agents", runCommands},
		{"auth", "manage credentials (login, import-mac, refresh, set-token, status)", runAuth},
		{"frames", "list / inspect / set-default frame", runFrames},
		{"frame", "alias for frames", runFrames},
		{"categories", "list categories on a frame", runCategories},
		{"category", "alias for categories", runCategories},
		{"chores", "list, create, update, claim, complete, delete chores", runChores},
		{"chore", "alias for chores", runChores},
		{"rewards", "list, create, update, redeem/delete rewards; show point balances", runRewards},
		{"reward", "alias for rewards", runRewards},
		{"calendar", "calendar events and source calendars", runCalendar},
		{"lists", "lists and list item management", runLists},
		{"list", "alias for lists", runLists},
		{"grocery", "grocery list helpers", runGrocery},
		{"meals", "meal categories, recipes, sittings, and grocery sync", runMeals},
		{"meal", "alias for meals", runMeals},
		{"photos", "photo list, upload, download, and delete", runPhotos},
		{"photo", "alias for photos", runPhotos},
		{"albums", "photo album reads", runAlbums},
		{"album", "alias for albums", runAlbums},
		{"routines", "routine list/create/update/delete/reorder", runRoutines},
		{"routine", "alias for routines", runRoutine},
		{"sidekick", "inspect Plus access and Sidekick history", runSidekick},
		{"bounties", "paired chore/reward bounty helpers", runBounties},
		{"bounty", "alias for bounties", runBounty},
		{"rotations", "create rotating chore schedules", runRotations},
		{"rotation", "alias for rotations", runRotation},
		{"status", "quick overview of the connected frame", runStatus},
		{"analytics", "family activity statistics", runAnalytics},
		{"home", "weekly combined events/tasks/lists view", runHome},
		{"watch", "poll for newly completed chores, redeemed rewards, and upcoming events", runWatch},
		{"export", "export frame data to JSON", runExport},
		{"import", "import frame data from a skycli export", runImport},
		{"config", "show or update skycli configuration", runConfig},
		{"raw", "send a raw HTTP request to the Skylight API", runRaw},
		{"version", "print version", runVersion},
	}
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	args = normalizeRootFlags(args)
	jsonRequested := hasRootFlag(args, "json")
	g := &globals{}
	root := flag.NewFlagSet("skycli", flag.ContinueOnError)
	if jsonRequested {
		root.SetOutput(io.Discard)
	} else {
		root.SetOutput(stderr)
	}
	g.register(root)
	root.Usage = func() {
		if !jsonRequested {
			printRootUsage(stderr, root)
		}
	}
	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if g.asJSON {
				_ = newPrinter(stdout, true, false).JSON(buildCommandCatalog())
			}
			return exitOK
		}
		if g.asJSON {
			_ = newPrinter(stdout, true, false).JSON(map[string]any{
				"error":     err.Error(),
				"kind":      "usage",
				"exit_code": exitUsage,
			})
		}
		return exitUsage
	}
	if os.Getenv("SKYCLI_READONLY") == "1" || strings.EqualFold(os.Getenv("SKYCLI_READONLY"), "true") {
		g.readOnly = true
	}
	if g.allow == "" {
		g.allow = os.Getenv("SKYCLI_ALLOW_COMMANDS")
	}
	if g.deny == "" {
		g.deny = os.Getenv("SKYCLI_DENY_COMMANDS")
	}
	if g.asJSON && g.plain {
		fmt.Fprintln(stderr, "choose only one of --json or --plain")
		return exitUsage
	}
	rest := root.Args()
	if g.doctor {
		if len(rest) > 0 {
			if g.asJSON {
				_ = newPrinter(stdout, true, false).JSON(map[string]any{
					"error":     "--doctor cannot be combined with a command",
					"kind":      "usage",
					"exit_code": exitUsage,
				})
			} else {
				fmt.Fprintln(stderr, "--doctor cannot be combined with a command")
			}
			return exitUsage
		}
		cfg, err := config.Load(g.configPath)
		if err != nil {
			if g.asJSON {
				_ = newPrinter(stdout, true, false).JSON(map[string]any{"error": err.Error()})
			} else {
				fmt.Fprintln(stderr, "error:", err)
			}
			return exitErr
		}
		rc := &runCtx{
			ctx:    ctx,
			stdin:  stdin,
			stdout: stdout,
			stderr: stderr,
			g:      g,
			cfg:    cfg,
			out:    newPrinter(stdout, g.asJSON, g.plain),
		}
		return runDoctor(rc, nil)
	}
	if len(rest) == 0 {
		if g.asJSON {
			_ = newPrinter(stdout, true, false).JSON(buildCommandCatalog())
			return exitOK
		}
		root.Usage()
		return exitUsage
	}
	cmd, sub := rest[0], rest[1:]

	if err := enforceSafety(g, rest); err != nil {
		fmt.Fprintln(stderr, "error:", err.Error())
		return exitErr
	}

	cfg, err := config.Load(g.configPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitErr
	}

	rc := &runCtx{
		ctx:    ctx,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		g:      g,
		cfg:    cfg,
		out:    newPrinter(stdout, g.asJSON, g.plain),
	}

	for _, c := range topLevelCommands() {
		if c.name == cmd {
			return c.run(rc, sub)
		}
	}
	if g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"error":     "unknown command: " + cmd,
			"kind":      "usage",
			"command":   cmd,
			"available": commandNames(),
			"exit_code": exitUsage,
		})
		return exitUsage
	}
	fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
	root.Usage()
	return exitUsage
}

// fail reports a human-readable error and returns exitErr.
func fail(rc *runCtx, err error) int {
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintln(rc.stderr, "error:", err.Error())
	}
	return exitErr
}

// usage prints help to stderr and returns exitUsage.
func usage(rc *runCtx, msg string) int {
	if msg != "" {
		if rc.g.asJSON {
			_ = rc.out.JSON(map[string]any{
				"error":     msg,
				"kind":      "usage",
				"exit_code": exitUsage,
			})
		} else {
			fmt.Fprintln(rc.stderr, msg)
		}
	}
	return exitUsage
}

func printRootUsage(w io.Writer, root *flag.FlagSet) {
	fmt.Fprintln(w, "skycli — unofficial CLI for the Skylight Calendar private API")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage: skycli [global flags] <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	root.PrintDefaults()
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	for _, c := range topLevelCommands() {
		fmt.Fprintf(w, "  %-12s %s\n", c.name, c.summary)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Env: SKYLIGHT_ACCESS_TOKEN, SKYLIGHT_AUTH_SCHEME, SKYLIGHT_FRAME_ID")
}

var rootBoolFlags = map[string]bool{
	"doctor":     true,
	"json":       true,
	"plain":      true,
	"readonly":   true,
	"trace-http": true,
}

var rootValueFlags = map[string]bool{
	"allow-commands": true,
	"config":         true,
	"deny-commands":  true,
	"frame":          true,
	"timeout":        true,
	"token":          true,
}

func normalizeRootFlags(args []string) []string {
	if len(args) == 0 {
		return args
	}
	global := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		name, ok, hasInlineValue := splitFlagName(arg)
		if !ok {
			rest = append(rest, arg)
			continue
		}
		if rootBoolFlags[name] {
			global = append(global, arg)
			continue
		}
		if rootValueFlags[name] {
			global = append(global, arg)
			if !hasInlineValue && i+1 < len(args) {
				i++
				global = append(global, args[i])
			}
			continue
		}
		rest = append(rest, arg)
	}
	return append(global, rest...)
}

func hasRootFlag(args []string, want string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		name, ok, _ := splitFlagName(arg)
		if ok && name == want {
			return true
		}
	}
	return false
}

func splitFlagName(arg string) (string, bool, bool) {
	if arg == "" || arg == "-" || !strings.HasPrefix(arg, "-") {
		return "", false, false
	}
	name := strings.TrimLeft(arg, "-")
	if name == "" {
		return "", false, false
	}
	if idx := strings.IndexByte(name, '='); idx >= 0 {
		return name[:idx], true, true
	}
	return name, true, false
}

func commandNames() []string {
	commands := topLevelCommands()
	names := make([]string, 0, len(commands))
	for _, c := range commands {
		names = append(names, c.name)
	}
	return names
}

func parseInt64Flag(s, name string) (int64, error) {
	if strings.TrimSpace(s) == "" {
		return 0, fmt.Errorf("--%s is required", name)
	}
	return skylight.ParseID(s)
}
