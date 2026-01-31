// internal/config/keyring.go
package config

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "ezdb"

// KeyringStore manages password storage in system keyring
type KeyringStore struct {
	ring keyring.Keyring
}

// NewKeyringStore creates a new keyring store instance
func NewKeyringStore() (*KeyringStore, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}
	return &KeyringStore{ring: ring}, nil
}

// SetPassword stores a password for a profile
func (k *KeyringStore) SetPassword(profileName, password string) error {
	return k.ring.Set(keyring.Item{
		Key:  profileName,
		Data: []byte(password),
	})
}

// GetPassword retrieves a password for a profile
func (k *KeyringStore) GetPassword(profileName string) (string, error) {
	item, err := k.ring.Get(profileName)
	if err != nil {
		return "", fmt.Errorf("password not found for profile: %s", profileName)
	}
	return string(item.Data), nil
}

// DeletePassword removes a password for a profile
func (k *KeyringStore) DeletePassword(profileName string) error {
	return k.ring.Remove(profileName)
}
