package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	key := []byte("test-secret")
	tok := mint(key, "root", time.Now())
	uid, ok := verify(key, tok, time.Now())
	if !ok {
		t.Fatalf("verify failed")
	}
	if uid != "root" {
		t.Fatalf("uid = %q, want root", uid)
	}
}

func TestTokenExpiry(t *testing.T) {
	key := []byte("test-secret")
	now := time.Now()
	tok := mint(key, "root", now)
	if _, ok := verify(key, tok, now.Add(tokenTTL).Add(time.Hour)); ok {
		t.Fatalf("expired token verified as valid")
	}
}

func TestTokenTamper(t *testing.T) {
	key := []byte("test-secret")
	tok := mint(key, "root", time.Now())
	if _, ok := verify([]byte("other"), tok, time.Now()); ok {
		t.Fatalf("token signed with different key verified")
	}
	if _, ok := verify(key, "AAAA.BBB", time.Now()); ok {
		t.Fatalf("garbage token verified")
	}
}

func TestAuthenticator(t *testing.T) {
	a := New("root:devpassword", "fixed-token", "")
	if tok, ok := a.Login("root:devpassword"); !ok || tok == "" {
		t.Fatalf("login failed")
	}
	if _, ok := a.Login("root:wrong"); ok {
		t.Fatalf("login with wrong password succeeded")
	}

	if !a.Allow("Bearer fixed-token") {
		t.Fatalf("fixed token not allowed")
	}
	tok, _ := a.Login("root:devpassword")
	if !a.Allow("Bearer " + tok) {
		t.Fatalf("login token not allowed")
	}
	if a.Allow("Bearer nope") {
		t.Fatalf("unknown token allowed")
	}
}

func TestDerivedSecretStable(t *testing.T) {
	a := New("root:devpassword", "fixed-token", "")
	tok, _ := a.Login("root:devpassword")
	a2 := New("root:devpassword", "fixed-token", "")
	if !a2.Allow("Bearer " + tok) {
		t.Fatalf("token must be stable across Authenticator instances (derived secret)")
	}
}