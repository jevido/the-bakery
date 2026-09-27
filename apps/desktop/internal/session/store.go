// Package session keeps the signed-in member's token between launches: in
// the OS keyring when there is one, otherwise in a 0600 file in the user's
// config directory.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const keyringService = "the-bakery"

// Store holds one token per API URL. Load returns "" when there is none.
type Store interface {
	Load() (string, error)
	Save(token string) error
	Delete() error
}

// NewStore returns a keyring store for apiURL, or a file store when the OS
// has no usable keyring (e.g. Linux without a Secret Service).
func NewStore(apiURL string, logger *slog.Logger) Store {
	k := keyringStore{user: apiURL}
	if _, err := keyring.Get(keyringService, k.user); err == nil || errors.Is(err, keyring.ErrNotFound) {
		return k
	} else {
		dir, derr := os.UserConfigDir()
		if derr != nil {
			dir = os.TempDir()
		}
		sum := sha256.Sum256([]byte(apiURL))
		path := filepath.Join(dir, "the-bakery", "token-"+hex.EncodeToString(sum[:8]))
		logger.Warn("no OS keyring available; keeping the token in a file instead", "path", path, "err", err)
		return fileStore{path: path}
	}
}

type keyringStore struct{ user string }

func (k keyringStore) Load() (string, error) {
	token, err := keyring.Get(keyringService, k.user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return token, err
}

func (k keyringStore) Save(token string) error { return keyring.Set(keyringService, k.user, token) }

func (k keyringStore) Delete() error {
	err := keyring.Delete(keyringService, k.user)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

type fileStore struct{ path string }

func (f fileStore) Load() (string, error) {
	b, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

func (f fileStore) Save(token string) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(f.path), err)
	}
	return os.WriteFile(f.path, []byte(token), 0o600)
}

func (f fileStore) Delete() error {
	err := os.Remove(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
