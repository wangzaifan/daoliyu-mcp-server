package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type AuthToken struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token"`
	Scopes    []string `json:"scopes"`
	CreatedAt string   `json:"createdAt"`
	Admin     bool     `json:"admin,omitempty"`
}

type TokenStore struct {
	Path   string
	mu     sync.RWMutex
	Tokens []AuthToken
}

func NewTokenStore(dir, legacyToken string) (*TokenStore, error) {
	t := &TokenStore{Path: filepath.Join(dir, "auth-tokens.json")}
	b, err := os.ReadFile(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		t.Tokens = []AuthToken{{ID: "legacy-admin", Name: "系统管理令牌", Token: legacyToken, Scopes: []string{"*"}, CreatedAt: time.Now().UTC().Format(time.RFC3339), Admin: true}}
		return t, t.save()
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &t.Tokens); err != nil {
		return nil, fmt.Errorf("read auth tokens: %w", err)
	}
	if len(t.Tokens) == 0 {
		t.Tokens = []AuthToken{{ID: "legacy-admin", Name: "系统管理令牌", Token: legacyToken, Scopes: []string{"*"}, CreatedAt: time.Now().UTC().Format(time.RFC3339), Admin: true}}
		if err := t.save(); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (t *TokenStore) List() []AuthToken {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]AuthToken, 0, len(t.Tokens))
	for _, token := range t.Tokens {
		if !token.Admin {
			out = append(out, token)
		}
	}
	return out
}
func (t *TokenStore) Find(value string) (AuthToken, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, token := range t.Tokens {
		if token.Token == value {
			return token, true
		}
	}
	return AuthToken{}, false
}
func (t *TokenStore) IsAdmin(value string) bool { token, ok := t.Find(value); return ok && token.Admin }
func (t *TokenStore) Create(name string, scopes []string) (AuthToken, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return AuthToken{}, fmt.Errorf("token name is required")
	}
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}
	value, err := randomToken()
	if err != nil {
		return AuthToken{}, err
	}
	id, err := randomID()
	if err != nil {
		return AuthToken{}, err
	}
	token := AuthToken{ID: id, Name: name, Token: value, Scopes: unique(scopes), CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Tokens = append(t.Tokens, token)
	if err := t.saveLocked(); err != nil {
		t.Tokens = t.Tokens[:len(t.Tokens)-1]
		return AuthToken{}, err
	}
	return token, nil
}
func (t *TokenStore) Delete(id string) error {
	if id == "legacy-admin" {
		return fmt.Errorf("系统管理令牌不能删除")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, token := range t.Tokens {
		if token.ID == id {
			previous := append([]AuthToken(nil), t.Tokens...)
			t.Tokens = append(t.Tokens[:i], t.Tokens[i+1:]...)
			if err := t.saveLocked(); err != nil {
				t.Tokens = previous
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("token not found")
}
func (t *TokenStore) Allowed(token AuthToken, pluginID string) bool {
	for _, scope := range token.Scopes {
		if scope == "*" || scope == pluginID {
			return true
		}
	}
	return false
}
func (t *TokenStore) save() error { t.mu.Lock(); defer t.mu.Unlock(); return t.saveLocked() }
func (t *TokenStore) saveLocked() error {
	b, err := json.MarshalIndent(t.Tokens, "", "  ")
	if err != nil {
		return err
	}
	tmp := t.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, t.Path)
}
func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "dly_" + hex.EncodeToString(b), nil
}
func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "tok_" + hex.EncodeToString(b), nil
}
func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
