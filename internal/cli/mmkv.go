package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// AuthStorage matches the JSON blob the Skylight iOS app persists in MMKV
// at ~/Library/Containers/com.skylightframe.mobile/Data/Documents/mmkv/mmkv.default.
type AuthStorage struct {
	UserID              string `json:"userId"`
	AccessToken         string `json:"accessToken"`
	RefreshToken        string `json:"refreshToken"`
	AccessTokenLifeSpan int64  `json:"accessTokenLifeSpan"`
	AccessTokenExpiry   int64  `json:"accessTokenExpiry"`
	UniqueID            string `json:"uniqueId"`
}

type authBlob struct {
	State AuthStorage `json:"state"`
}

func DefaultMMKVPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Containers", "com.skylightframe.mobile",
		"Data", "Documents", "mmkv", "mmkv.default")
}

// blobRe matches the auth-storage JSON object inside the MMKV byte stream.
// MMKV stores values as length-prefixed strings; the JSON itself is intact.
var blobRe = regexp.MustCompile(`\{"state":\{[^}]*"accessToken"[^}]*\},"version":\d+\}`)

func ReadMMKVAuth(path string) (*AuthStorage, error) {
	if path == "" {
		path = DefaultMMKVPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mmkv %s: %w", path, err)
	}
	matches := blobRe.FindAll(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no auth-storage blob found in %s", path)
	}
	var best *AuthStorage
	for _, m := range matches {
		var b authBlob
		if err := json.Unmarshal(m, &b); err != nil {
			continue
		}
		s := b.State
		if s.AccessToken == "" {
			continue
		}
		if best == nil {
			best = &s
			continue
		}
		// Prefer entries with a real userId, then prefer later expiry.
		bestHasUser := best.UserID != ""
		curHasUser := s.UserID != ""
		switch {
		case curHasUser && !bestHasUser:
			best = &s
		case curHasUser == bestHasUser && s.AccessTokenExpiry > best.AccessTokenExpiry:
			best = &s
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no entry with a non-empty accessToken in %s", path)
	}
	return best, nil
}

func (a *AuthStorage) ExpiresAt() time.Time {
	if a.AccessTokenExpiry == 0 {
		return time.Time{}
	}
	return time.UnixMilli(a.AccessTokenExpiry)
}
