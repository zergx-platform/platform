package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// tokenTTL is how long a minted login token stays valid. Long enough that the
// client "fills in the credential once" and keeps the signed token cached
// locally (SharedPreferences / localStorage); short enough that a leaked token
// is not forever.
const tokenTTL = 30 * 24 * time.Hour

type tokenPayload struct {
	Uid string `json:"uid"`
	Exp int64  `json:"exp"`
}

// mint produces a compact HMAC-signed token: base64url(payload).base64url(mac).
// The payload is public; integrity and authenticity come from the MAC.
func mint(key []byte, uid string, now time.Time) string {
	payload, _ := json.Marshal(tokenPayload{Uid: uid, Exp: now.Add(tokenTTL).Unix()})
	enc := base64.RawURLEncoding.EncodeToString(payload)
	mac := sign(key, enc)
	return enc + "." + base64.RawURLEncoding.EncodeToString(mac)
}

// verify decrypts the token if and only if the MAC is valid and it has not
// expired. Returns the uid, or ok=false.
func verify(key []byte, token string, now time.Time) (string, bool) {
	dot := lastDot(token)
	if dot <= 0 || dot == len(token)-1 {
		return "", false
	}
	enc, macEnc := token[:dot], token[dot+1:]
	mac, err := base64.RawURLEncoding.DecodeString(macEnc)
	if err != nil {
		return "", false
	}
	if !hmac.Equal(mac, sign(key, enc)) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", false
	}
	var p tokenPayload
	if json.Unmarshal(raw, &p) != nil {
		return "", false
	}
	if p.Exp < now.Unix() {
		return "", false
	}
	return p.Uid, true
}

func sign(key []byte, msg string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(msg))
	return h.Sum(nil)
}

// lastDot returns the index of the last '.' that is not the final character
// (the MAC base64url must be non-empty), or -1 when absent.
func lastDot(s string) int {
	i := -1
	for j := len(s) - 1; j >= 0; j-- {
		if s[j] == '.' {
			i = j
			break
		}
	}
	return i
}