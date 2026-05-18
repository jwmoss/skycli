package cli

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jwmoss/skycli/internal/config"
)

const (
	secretsBackendConfig   = "config"
	secretsBackendKeychain = "keychain"
	secretsBackendFile     = "file"
	secretsEnvFileKey      = "SKYCLI_FILE_SECRET_KEY"
)

type storedSecrets struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func defaultSecretsBackend() string {
	if runtime.GOOS == "darwin" {
		return secretsBackendKeychain
	}
	if os.Getenv(secretsEnvFileKey) != "" {
		return secretsBackendFile
	}
	return secretsBackendConfig
}

func (rc *runCtx) secretsBackend() string {
	if rc.cfg.SecretsBackend != "" {
		return rc.cfg.SecretsBackend
	}
	return defaultSecretsBackend()
}

func (rc *runCtx) loadConfiguredSecrets() {
	if rc.secretsLoaded {
		return
	}
	rc.secretsLoaded = true
	backend := rc.secretsBackend()
	if backend == secretsBackendConfig {
		return
	}
	secrets, err := rc.readSecrets(backend)
	if err == nil {
		if secrets.AccessToken != "" {
			rc.cfg.AccessToken = secrets.AccessToken
		}
		if secrets.RefreshToken != "" {
			rc.cfg.RefreshToken = secrets.RefreshToken
		}
	}
	if (rc.cfg.AccessToken != "" || rc.cfg.RefreshToken != "") && secrets.AccessToken == "" && secrets.RefreshToken == "" {
		if err := rc.saveConfiguredSecrets(); err != nil {
			fmt.Fprintf(rc.stderr, "warning: migrate secrets to %s: %v\n", backend, err)
			return
		}
		rc.cfg.AccessToken = ""
		rc.cfg.RefreshToken = ""
		_ = rc.saveConfig()
		_ = rc.loadSecretsIntoConfig()
	}
}

func (rc *runCtx) loadSecretsIntoConfig() error {
	if rc.secretsBackend() == secretsBackendConfig {
		return nil
	}
	secrets, err := rc.readSecrets(rc.secretsBackend())
	if err != nil {
		return err
	}
	rc.cfg.AccessToken = secrets.AccessToken
	rc.cfg.RefreshToken = secrets.RefreshToken
	return nil
}

func (rc *runCtx) saveConfiguredSecrets() error {
	backend := rc.secretsBackend()
	if backend == secretsBackendConfig {
		return nil
	}
	return rc.writeSecrets(backend, storedSecrets{
		AccessToken:  rc.cfg.AccessToken,
		RefreshToken: rc.cfg.RefreshToken,
	})
}

func (rc *runCtx) saveConfig() error {
	cfgForDisk := *rc.cfg
	if rc.secretsBackend() != secretsBackendConfig {
		cfgForDisk.AccessToken = ""
		cfgForDisk.RefreshToken = ""
	}
	return config.Save(rc.g.configPath, &cfgForDisk)
}

func (rc *runCtx) readSecrets(backend string) (storedSecrets, error) {
	switch backend {
	case secretsBackendConfig:
		return storedSecrets{AccessToken: rc.cfg.AccessToken, RefreshToken: rc.cfg.RefreshToken}, nil
	case secretsBackendKeychain:
		return readKeychainSecrets()
	case secretsBackendFile:
		return readFileSecrets()
	default:
		return storedSecrets{}, fmt.Errorf("unknown secrets backend %q", backend)
	}
}

func (rc *runCtx) writeSecrets(backend string, secrets storedSecrets) error {
	switch backend {
	case secretsBackendConfig:
		rc.cfg.AccessToken = secrets.AccessToken
		rc.cfg.RefreshToken = secrets.RefreshToken
		return nil
	case secretsBackendKeychain:
		return writeKeychainSecrets(secrets)
	case secretsBackendFile:
		return writeFileSecrets(secrets)
	default:
		return fmt.Errorf("unknown secrets backend %q", backend)
	}
}

func keychainService() string {
	return "skycli"
}

func readKeychainSecrets() (storedSecrets, error) {
	access, accessErr := keychainRead(keychainService(), "access_token")
	refresh, refreshErr := keychainRead(keychainService(), "refresh_token")
	if accessErr != nil && refreshErr != nil {
		return storedSecrets{}, accessErr
	}
	return storedSecrets{AccessToken: access, RefreshToken: refresh}, nil
}

func writeKeychainSecrets(secrets storedSecrets) error {
	service := keychainService()
	if err := keychainWrite(service, "access_token", secrets.AccessToken); err != nil {
		return err
	}
	if err := keychainWrite(service, "refresh_token", secrets.RefreshToken); err != nil {
		return err
	}
	return nil
}

func keychainRead(service, account string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("keychain backend requires macOS; set secrets_backend=file or config")
	}
	out, err := exec.Command("/usr/bin/security", "find-generic-password", "-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("read keychain %s/%s: %w", service, account, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func keychainWrite(service, account, value string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("keychain backend requires macOS; set secrets_backend=file or config")
	}
	_ = exec.Command("/usr/bin/security", "delete-generic-password", "-s", service, "-a", account).Run()
	if value == "" {
		return nil
	}
	cmd := exec.Command("/usr/bin/security", "add-generic-password", "-s", service, "-a", account, "-w", value, "-U")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write keychain %s/%s: %w: %s", service, account, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func fileSecretsPath() (string, error) {
	root, err := config.RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "secrets.json.enc"), nil
}

func readFileSecrets() (storedSecrets, error) {
	path, err := fileSecretsPath()
	if err != nil {
		return storedSecrets{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return storedSecrets{}, err
	}
	plain, err := decryptFileSecret(data)
	if err != nil {
		return storedSecrets{}, err
	}
	var secrets storedSecrets
	if err := json.Unmarshal(plain, &secrets); err != nil {
		return storedSecrets{}, err
	}
	return secrets, nil
}

func writeFileSecrets(secrets storedSecrets) error {
	path, err := fileSecretsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	encrypted, err := encryptFileSecret(plain)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encrypted, 0o600)
}

func fileSecretKey() ([]byte, error) {
	raw := os.Getenv(secretsEnvFileKey)
	if raw == "" {
		return nil, fmt.Errorf("%s is required for file secret storage", secretsEnvFileKey)
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func encryptFileSecret(plain []byte) ([]byte, error) {
	key, err := fileSecretKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := append([]byte("skycli-secret-v1:"), nonce...)
	out = gcm.Seal(out, nonce, plain, nil)
	return out, nil
}

func decryptFileSecret(data []byte) ([]byte, error) {
	const prefix = "skycli-secret-v1:"
	if !strings.HasPrefix(string(data), prefix) {
		return nil, errors.New("unknown secret file format")
	}
	key, err := fileSecretKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	payload := data[len(prefix):]
	if len(payload) < gcm.NonceSize() {
		return nil, errors.New("secret file is truncated")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
