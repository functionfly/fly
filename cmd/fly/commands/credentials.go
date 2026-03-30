package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "functionfly"
	keyringUser    = "cli-credentials"
)

// Credentials stores the authenticated user's identity and token.
type Credentials struct {
	Version      string    `json:"version"`
	User         UserInfo  `json:"user"`
	Token        string    `json:"token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserInfo holds the authenticated user's profile.
type UserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Provider  string `json:"provider"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".functionfly", "credentials.json"), nil
}

// LoadCredentials reads credentials from the OS keychain, falling back to the file on disk.
func LoadCredentials() (*Credentials, error) {
	// Try OS keychain first
	data, err := keyring.Get(keyringService, keyringUser)
	if err == nil {
		var creds Credentials
		if err := json.Unmarshal([]byte(data), &creds); err == nil {
			if !creds.ExpiresAt.IsZero() && time.Now().After(creds.ExpiresAt) {
				return nil, fmt.Errorf("your session has expired\n   → Run: fly login")
			}
			return &creds, nil
		}
	}

	// Fall back to file-based storage
	return loadCredentialsFromFile()
}

// loadCredentialsFromFile reads credentials from disk (legacy fallback).
func loadCredentialsFromFile() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in\n   → Run: fly login")
		}
		return nil, fmt.Errorf("could not read credentials: %w", err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("credentials file is corrupted\n   → Run: fly login")
	}
	if !creds.ExpiresAt.IsZero() && time.Now().After(creds.ExpiresAt) {
		return nil, fmt.Errorf("your session has expired\n   → Run: fly login")
	}
	return &creds, nil
}

// SaveCredentials writes credentials to the OS keychain, with a file-based fallback.
func SaveCredentials(creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("could not serialize credentials: %w", err)
	}

	// Try OS keychain first
	if err := keyring.Set(keyringService, keyringUser, string(data)); err != nil {
		// Keychain unavailable (headless Linux, CI, etc.) — fall back to file
		if err := saveCredentialsToFile(creds); err != nil {
			return err
		}
		return nil
	}

	// Keychain write succeeded — remove any legacy file to avoid confusion
	_ = deleteCredentialsFile()
	return nil
}

// saveCredentialsToFile writes credentials to disk (legacy fallback).
func saveCredentialsToFile(creds *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("could not create credentials directory: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("could not serialize credentials: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// DeleteCredentials removes credentials from both the OS keychain and disk.
func DeleteCredentials() error {
	// Remove from keychain (ignore if not found)
	_ = keyring.Delete(keyringService, keyringUser)
	// Remove legacy file
	return deleteCredentialsFile()
}

// deleteCredentialsFile removes the credentials file from disk.
func deleteCredentialsFile() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove credentials: %w", err)
	}
	return nil
}

// IsLoggedIn returns true if valid credentials exist.
func IsLoggedIn() bool {
	_, err := LoadCredentials()
	return err == nil
}

// resolveAuthorName returns (author, name) either from an optional positional
// "author/name" argument or by reading functionfly.jsonc + stored credentials.
func resolveAuthorName(args []string) (author, name string, err error) {
	if len(args) > 0 {
		parts := splitAuthorName(args[0])
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid argument %q — expected author/name", args[0])
		}
		return parts[0], parts[1], nil
	}
	manifest, merr := LoadManifest("")
	if merr != nil {
		return "", "", fmt.Errorf("no functionfly.jsonc found — run 'fly init' or pass author/name as argument")
	}
	creds, cerr := LoadCredentials()
	if cerr != nil {
		return "", "", fmt.Errorf("not logged in — run 'fly login'")
	}
	return creds.User.Username, manifest.Name, nil
}

func splitAuthorName(s string) []string {
	for i, c := range s {
		if c == '/' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
