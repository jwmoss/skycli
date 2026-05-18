package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMMKVAuthDoesNotDependOnUserIDFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mmkv.default")
	data := []byte(`prefix {"state":{"accessToken":"access","refreshToken":"refresh","uniqueId":"fp","accessTokenExpiry":1234567890,"userId":"user"},"version":0} suffix`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	auth, err := ReadMMKVAuth(path)
	if err != nil {
		t.Fatalf("ReadMMKVAuth: %v", err)
	}
	if auth.AccessToken != "access" || auth.RefreshToken != "refresh" || auth.UniqueID != "fp" {
		t.Fatalf("auth = %+v", auth)
	}
}
