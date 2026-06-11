package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// acquireLockFile opens (creating if needed) path and takes an exclusive
// advisory lock, blocking until it is available. The returned func releases
// the lock; the lock file itself is left in place for reuse.
func acquireLockFile(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create lock dir for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return func() {
		_ = unlockFile(f)
		_ = f.Close()
	}, nil
}
