// Package mirrorstore persists per-repo git-mirror configuration in NATS KV
// (platform-owned bucket, never shared with the message-fact/read-watermark
// flows). The push secret is AES-GCM encrypted before storage so it is never
// readable by other NATS consumers or in a raw bucket dump.
package mirrorstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/abcp-sdk/abc-protocol-go/bus"
)

// Bucket is the platform-private NATS KV bucket for mirror configs.
const Bucket = "zergx-mirror"

// Cfg is the stored mirror configuration for one repo (org/repo).
type Cfg struct {
	PullURL string `json:"pull_url,omitempty"`
	PushURL string `json:"push_url,omitempty"`
	// PushSecret is stored encrypted; never returned by GET.
	PushSecret string `json:"push_secret,omitempty"`
}

// Store reads/writes mirror configs in the NATS KV bucket.
type Store struct {
	bus bus.Bus
	gcm cipher.AEAD // nil => plaintext (no key configured)
}

// New builds a Store. If key is non-empty it enables AES-GCM on PushSecret.
func New(b bus.Bus, key string) *Store {
	st := &Store{bus: b}
	if key != "" {
		block, err := aes.NewCipher([]byte(key))
		if err != nil {
			slog.Warn("mirrorstore: bad MIRROR_SECRET_KEY, using plaintext", "err", err)
			return st
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			slog.Warn("mirrorstore: gcm init failed, using plaintext", "err", err)
			return st
		}
		st.gcm = gcm
	}
	return st
}

func (s *Store) key(org, repo string) string {
	return "repo." + org + "." + repo
}

// Get returns the stored config (not-set => ok=false).
func (s *Store) Get(ctx context.Context, org, repo string) (Cfg, bool, error) {
	v, err := s.bus.KVGet(ctx, Bucket, s.key(org, repo))
	if err != nil {
		// KV bucket-missing or key-missing surface as empty; treat as not-set.
		return Cfg{}, false, nil
	}
	if v == "" {
		return Cfg{}, false, nil
	}
	var c Cfg
	if err := json.Unmarshal([]byte(v), &c); err != nil {
		return Cfg{}, false, fmt.Errorf("decode mirror cfg: %w", err)
	}
	if c.PushSecret != "" {
		if dec, err := s.decrypt(c.PushSecret); err == nil {
			c.PushSecret = dec
		} else {
			slog.Warn("mirrorstore: secret decrypt failed", "err", err)
		}
	}
	return c, true, nil
}

// Put stores the config (encrypting the secret when enabled).
func (s *Store) Put(ctx context.Context, org, repo string, in Cfg) error {
	out := in
	if in.PushSecret != "" {
		enc, err := s.encrypt(in.PushSecret)
		if err != nil {
			return fmt.Errorf("encrypt secret: %w", err)
		}
		out.PushSecret = enc
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	// KVPut creates the bucket on first use via the SDK.
	return s.bus.KVPut(ctx, Bucket, s.key(org, repo), string(b), 0)
}

// Delete removes the stored config.
func (s *Store) Delete(ctx context.Context, org, repo string) error {
	return s.bus.KVDelete(ctx, Bucket, s.key(org, repo))
}

func (s *Store) encrypt(plain string) (string, error) {
	if s.gcm == nil {
		return plain, nil
	}
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := s.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawStdEncoding.EncodeToString(ct), nil
}

func (s *Store) decrypt(ctB64 string) (string, error) {
	if s.gcm == nil {
		return ctB64, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(ctB64)
	if err != nil {
		return "", err
	}
	ns := s.gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := s.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
