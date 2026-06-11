package cli

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/config"
	"github.com/jwmoss/skycli/internal/skylight"
	"golang.org/x/term"
)

type fdReader interface {
	Fd() uintptr
}

var (
	stdinIsTerminal    = term.IsTerminal
	readTerminalSecret = term.ReadPassword
)

func runAuth(rc *runCtx, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(rc.stderr, "skycli auth <login|import-mac|refresh|set-token|status>")
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "login":
		return authLogin(rc, rest)
	case "import-mac":
		return authImportMac(rc, rest)
	case "refresh":
		return authRefresh(rc, rest)
	case "set-token":
		return authSetToken(rc, rest)
	case "status":
		return authStatus(rc, rest)
	default:
		return usage(rc, fmt.Sprintf("unknown auth subcommand: %s", sub))
	}
}

func authLogin(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	email := fs.String("email", "", "Skylight account email")
	fingerprint := fs.String("fingerprint", "", "device fingerprint UUID; generated when omitted")
	passwordStdin := fs.Bool("password-stdin", false, "read password from stdin")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*email, "email"); err != nil {
		return usage(rc, err.Error())
	}
	password, err := readLoginPassword(rc, *passwordStdin)
	if err != nil {
		return fail(rc, err)
	}
	if strings.TrimSpace(password) == "" {
		return fail(rc, fmt.Errorf("empty password"))
	}
	fp := strings.TrimSpace(*fingerprint)
	if fp == "" {
		fp = newUUID()
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
	tok, err := c.LoginOAuth(rc.ctx, *email, password, fp)
	if err != nil {
		return fail(rc, err)
	}
	expiresAt := time.Time{}
	if tok.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	rc.cfg.AccessToken = tok.AccessToken
	rc.cfg.RefreshToken = tok.RefreshToken
	rc.cfg.AccessTokenExpAt = expiresAt
	rc.cfg.DeviceFingerprint = fp
	rc.cfg.AuthScheme = config.DefaultAuthScheme
	if err := rc.saveConfiguredSecrets(); err != nil {
		return fail(rc, err)
	}
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	_ = rc.loadSecretsIntoConfig()
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"logged_in":             true,
			"expires_at":            expiresAt,
			"expires_in_seconds":    tok.ExpiresIn,
			"refresh_token_saved":   tok.RefreshToken != "",
			"fingerprint":           fp,
			"fingerprint_generated": *fingerprint == "",
		})
		return exitOK
	}
	if expiresAt.IsZero() {
		rc.out.Line("logged in; refresh token saved")
	} else {
		rc.out.Line("logged in; token expires_at: %s (in %s)",
			expiresAt.Format(time.RFC3339), time.Until(expiresAt).Round(time.Second))
	}
	rc.out.Line("device fingerprint: %s", fp)
	return exitOK
}

func authImportMac(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("auth import-mac", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	var path string
	fs.StringVar(&path, "mmkv", "", "path to mmkv.default (default: Skylight Mac container)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	auth, err := ReadMMKVAuth(path)
	if err != nil {
		return fail(rc, err)
	}
	rc.cfg.AccessToken = auth.AccessToken
	rc.cfg.RefreshToken = auth.RefreshToken
	rc.cfg.AccessTokenExpAt = auth.ExpiresAt()
	rc.cfg.DeviceFingerprint = auth.UniqueID
	rc.cfg.AuthScheme = config.DefaultAuthScheme
	if err := rc.saveConfiguredSecrets(); err != nil {
		return fail(rc, err)
	}
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	_ = rc.loadSecretsIntoConfig()
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"imported":            true,
			"user_id":             auth.UserID,
			"expires_at":          auth.ExpiresAt(),
			"refresh_token_saved": auth.RefreshToken != "",
			"fingerprint_saved":   auth.UniqueID != "",
		})
		return exitOK
	}
	exp := auth.ExpiresAt()
	rc.out.Line("imported token for user_id=%s", auth.UserID)
	rc.out.Line("expires_at: %s (in %s)", exp.Format(time.RFC3339), time.Until(exp).Round(time.Second))
	if auth.RefreshToken != "" {
		if auth.UniqueID != "" {
			rc.out.Line("refresh_token saved; future commands will auto-refresh when needed")
		} else {
			rc.out.Line("refresh_token saved, but no device fingerprint was found for auto-refresh")
		}
	}
	return exitOK
}

func readLoginPassword(rc *runCtx, passwordStdin bool) (string, error) {
	if passwordStdin {
		return readSingleLine(rc.stdin)
	}
	return readSecretLine(rc, "Password: ", "Paste password (single line), then press Enter:")
}

func readSecretLine(rc *runCtx, prompt, fallbackPrompt string) (string, error) {
	if f, ok := rc.stdin.(fdReader); ok {
		fd := int(f.Fd())
		if stdinIsTerminal(fd) {
			fmt.Fprint(rc.stderr, prompt)
			data, err := readTerminalSecret(fd)
			fmt.Fprintln(rc.stderr)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(data)), nil
		}
	}
	fmt.Fprintln(rc.stderr, fallbackPrompt)
	return readSingleLine(rc.stdin)
}

func readSingleLine(r io.Reader) (string, error) {
	rd := bufio.NewReader(r)
	line, err := rd.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("skycli-%x", now)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func authRefresh(rc *runCtx, args []string) int {
	rc.loadConfiguredSecrets()
	fs := flag.NewFlagSet("auth refresh", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	refreshToken := fs.String("refresh-token", "", "override refresh token for this refresh and save it on success")
	fingerprint := fs.String("fingerprint", "", "override device fingerprint for this refresh and save it on success")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *refreshToken != "" {
		rc.cfg.RefreshToken = *refreshToken
	}
	if *fingerprint != "" {
		rc.cfg.DeviceFingerprint = *fingerprint
	}
	tok, expiresAt, err := rc.refreshConfiguredToken(true)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"refreshed":           true,
			"expires_at":          expiresAt,
			"expires_in_seconds":  tok.ExpiresIn,
			"refresh_token_saved": rc.cfg.RefreshToken != "",
		})
		return exitOK
	}
	if expiresAt.IsZero() {
		rc.out.Line("refreshed access token")
	} else {
		rc.out.Line("refreshed access token; expires_at: %s (in %s)",
			expiresAt.Format(time.RFC3339), time.Until(expiresAt).Round(time.Second))
	}
	return exitOK
}

func authSetToken(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("auth set-token", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	var scheme string
	fs.StringVar(&scheme, "scheme", config.DefaultAuthScheme, "auth scheme: Bearer or Basic")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	token, err := readSecretLine(rc, "Token: ", "Paste token (single line), then press Enter:")
	if err != nil {
		return fail(rc, err)
	}
	if token == "" {
		return fail(rc, fmt.Errorf("empty token"))
	}
	// Allow pasting the full "Bearer xxx" header value.
	if i := strings.IndexByte(token, ' '); i > 0 {
		head := token[:i]
		switch strings.ToLower(head) {
		case "bearer":
			scheme = "Bearer"
			token = strings.TrimSpace(token[i+1:])
		case "basic":
			scheme = "Basic"
			token = strings.TrimSpace(token[i+1:])
		}
	}
	rc.cfg.AccessToken = token
	rc.cfg.AuthScheme = scheme
	rc.cfg.RefreshToken = ""
	rc.cfg.AccessTokenExpAt = time.Time{}
	rc.cfg.DeviceFingerprint = ""
	if err := rc.saveConfiguredSecrets(); err != nil {
		return fail(rc, err)
	}
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	_ = rc.loadSecretsIntoConfig()
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"saved": true, "scheme": scheme})
	} else {
		rc.out.Line("saved %s token", scheme)
	}
	return exitOK
}

func authStatus(rc *runCtx, args []string) int {
	_ = args
	rc.loadConfiguredSecrets()
	path := rc.g.configPath
	if path == "" {
		p, _ := config.DefaultPath()
		path = p
	}
	info := map[string]any{
		"config_path":            path,
		"secrets_backend":        rc.secretsBackend(),
		"has_token":              rc.cfg.AccessToken != "",
		"scheme":                 rc.cfg.AuthScheme,
		"base_url":               rc.cfg.BaseURL,
		"api_version":            rc.cfg.APIVersion,
		"frame_default":          rc.cfg.DefaultFrameID,
		"has_refresh_token":      rc.cfg.RefreshToken != "",
		"has_device_fingerprint": rc.cfg.DeviceFingerprint != "",
	}
	if !rc.cfg.AccessTokenExpAt.IsZero() {
		info["expires_at"] = rc.cfg.AccessTokenExpAt
		info["expires_in_seconds"] = int(time.Until(rc.cfg.AccessTokenExpAt).Seconds())
		info["expired"] = time.Now().After(rc.cfg.AccessTokenExpAt)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(info)
		return exitOK
	}
	rc.out.Line("config:    %s", path)
	rc.out.Line("base_url:  %s", rc.cfg.BaseURL)
	rc.out.Line("api_ver:   %s", rc.cfg.APIVersion)
	rc.out.Line("scheme:    %s", rc.cfg.AuthScheme)
	rc.out.Line("has_token: %s", boolYN(rc.cfg.AccessToken != ""))
	rc.out.Line("refresh:   %s", boolYN(rc.cfg.RefreshToken != ""))
	rc.out.Line("fingerprint: %s", boolYN(rc.cfg.DeviceFingerprint != ""))
	if !rc.cfg.AccessTokenExpAt.IsZero() {
		if time.Now().After(rc.cfg.AccessTokenExpAt) {
			rc.out.Line("expired:   %s ago — run `skycli auth refresh`, `skycli auth login`, or `skycli auth import-mac`",
				time.Since(rc.cfg.AccessTokenExpAt).Round(time.Second))
		} else {
			rc.out.Line("expires:   %s (in %s)",
				rc.cfg.AccessTokenExpAt.Format(time.RFC3339),
				time.Until(rc.cfg.AccessTokenExpAt).Round(time.Second))
		}
	}
	if rc.cfg.DefaultFrameID != 0 {
		rc.out.Line("frame:     %d (default)", rc.cfg.DefaultFrameID)
	}
	return exitOK
}
