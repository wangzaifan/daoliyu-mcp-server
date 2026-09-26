package app

import "testing"

func TestTokenStoreScopes(t *testing.T) {
	tokens, err := NewTokenStore(t.TempDir(), "legacy")
	if err != nil {
		t.Fatal(err)
	}
	token, err := tokens.Create("思源客户端", []string{"siyuan"})
	if err != nil {
		t.Fatal(err)
	}
	if !tokens.Allowed(token, "siyuan") || tokens.Allowed(token, "other") {
		t.Fatal("scope mismatch")
	}
	if !tokens.IsAdmin("legacy") {
		t.Fatal("legacy token should manage tokens")
	}
	if got := tokens.List(); len(got) != 1 || got[0].ID != token.ID {
		t.Fatal("call token list must not expose the system admin token")
	}
}
