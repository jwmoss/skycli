package cli

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jwmoss/skycli/internal/config"
)

var configKeys = map[string]func(*config.Config) *string{
	"base_url":           func(c *config.Config) *string { return &c.BaseURL },
	"api_version":        func(c *config.Config) *string { return &c.APIVersion },
	"auth_scheme":        func(c *config.Config) *string { return &c.AuthScheme },
	"secrets_backend":    func(c *config.Config) *string { return &c.SecretsBackend },
	"access_token":       func(c *config.Config) *string { return &c.AccessToken },
	"refresh_token":      func(c *config.Config) *string { return &c.RefreshToken },
	"device_fingerprint": func(c *config.Config) *string { return &c.DeviceFingerprint },
}

func runConfig(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return configShow(rc, nil)
	}
	switch args[0] {
	case "show":
		return configShow(rc, args[1:])
	case "get":
		return configGet(rc, args[1:])
	case "set":
		return configSet(rc, args[1:])
	case "unset":
		return configUnset(rc, args[1:])
	case "edit":
		return configEdit(rc, args[1:])
	default:
		return usage(rc, "unknown config subcommand: "+args[0])
	}
}

func configPath(rc *runCtx) string {
	if rc.g.configPath != "" {
		return rc.g.configPath
	}
	p, _ := config.DefaultPath()
	return p
}

func configShow(rc *runCtx, args []string) int {
	rc.loadConfiguredSecrets()
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	showSecrets := fs.Bool("show-secrets", false, "show token values without masking")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	info := map[string]any{
		"path":                 configPath(rc),
		"base_url":             rc.cfg.BaseURL,
		"api_version":          rc.cfg.APIVersion,
		"auth_scheme":          rc.cfg.AuthScheme,
		"secrets_backend":      rc.secretsBackend(),
		"default_frame_id":     rc.cfg.DefaultFrameID,
		"access_token":         maskConfigValue(rc.cfg.AccessToken, *showSecrets),
		"refresh_token":        maskConfigValue(rc.cfg.RefreshToken, *showSecrets),
		"device_fingerprint":   rc.cfg.DeviceFingerprint,
		"access_token_expires": rc.cfg.AccessTokenExpAt,
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(info)
		return exitOK
	}
	rc.out.Line("config:    %s", info["path"])
	rc.out.Line("base_url:  %s", rc.cfg.BaseURL)
	rc.out.Line("api_ver:   %s", rc.cfg.APIVersion)
	rc.out.Line("scheme:    %s", rc.cfg.AuthScheme)
	rc.out.Line("secrets:   %s", rc.secretsBackend())
	rc.out.Line("frame:     %d", rc.cfg.DefaultFrameID)
	rc.out.Line("token:     %s", info["access_token"])
	rc.out.Line("refresh:   %s", info["refresh_token"])
	rc.out.Line("fingerprint: %s", dashIfEmpty(rc.cfg.DeviceFingerprint))
	return exitOK
}

func configGet(rc *runCtx, args []string) int {
	if len(args) != 1 {
		return usage(rc, "skycli config get <key>")
	}
	key := normalizeConfigKey(args[0])
	if key == "access_token" || key == "refresh_token" {
		rc.loadConfiguredSecrets()
	}
	if key == "default_frame_id" {
		rc.out.Line("%d", rc.cfg.DefaultFrameID)
		return exitOK
	}
	getter, ok := configKeys[key]
	if !ok {
		return usage(rc, "unknown config key: "+args[0])
	}
	rc.out.Line("%s", *getter(rc.cfg))
	return exitOK
}

func configSet(rc *runCtx, args []string) int {
	if len(args) != 2 {
		return usage(rc, "skycli config set <key> <value>")
	}
	key := normalizeConfigKey(args[0])
	value := args[1]
	if key == "access_token" || key == "refresh_token" {
		rc.loadConfiguredSecrets()
	}
	if key == "default_frame_id" {
		id, err := skylightParseID(value)
		if err != nil {
			return fail(rc, err)
		}
		rc.cfg.DefaultFrameID = id
	} else {
		setter, ok := configKeys[key]
		if !ok {
			return usage(rc, "unknown config key: "+args[0])
		}
		*setter(rc.cfg) = value
	}
	if key == "access_token" || key == "refresh_token" {
		if err := rc.saveConfiguredSecrets(); err != nil {
			return fail(rc, err)
		}
	}
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	_ = rc.loadSecretsIntoConfig()
	rc.out.Line("set %s", key)
	return exitOK
}

func configUnset(rc *runCtx, args []string) int {
	if len(args) != 1 {
		return usage(rc, "skycli config unset <key>")
	}
	key := normalizeConfigKey(args[0])
	if key == "access_token" || key == "refresh_token" {
		rc.loadConfiguredSecrets()
	}
	if key == "default_frame_id" {
		rc.cfg.DefaultFrameID = 0
	} else {
		setter, ok := configKeys[key]
		if !ok {
			return usage(rc, "unknown config key: "+args[0])
		}
		*setter(rc.cfg) = ""
	}
	if key == "access_token" || key == "refresh_token" {
		if err := rc.saveConfiguredSecrets(); err != nil {
			return fail(rc, err)
		}
	}
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	_ = rc.loadSecretsIntoConfig()
	rc.out.Line("unset %s", key)
	return exitOK
}

func configEdit(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("config edit", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path := configPath(rc)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fail(rc, err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfgForDisk := *rc.cfg
		if rc.secretsBackend() != secretsBackendConfig {
			cfgForDisk.AccessToken = ""
			cfgForDisk.RefreshToken = ""
		}
		data, _ := json.MarshalIndent(cfgForDisk, "", "  ")
		data = append(data, '\n')
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return fail(rc, err)
		}
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fail(rc, err)
	}
	return exitOK
}

func normalizeConfigKey(key string) string {
	key = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(key)), "skylight_")
	key = strings.ReplaceAll(key, "-", "_")
	switch key {
	case "token":
		return "access_token"
	case "refresh":
		return "refresh_token"
	case "frame", "frame_id":
		return "default_frame_id"
	}
	return key
}

func maskConfigValue(value string, show bool) string {
	if value == "" {
		return "(not set)"
	}
	if show {
		return value
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:4] + "****"
}

func skylightParseID(value string) (int64, error) {
	return parseInt64Flag(value, "value")
}
