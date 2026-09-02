// Package sessionstate: platform-side view of per-session facts carried on
// the abc Bus.
//
// Two independent KV data flows meet here, both written by other services:
//
//   - Message facts (bucket `abc-session-state`): the agent mirrors the
//     newest-message timestamp/preview/role of each session at every persist
//     site. The agent knows nothing about read/unread — it projects message
//     data it already owns in PG.
//   - Read watermarks (bucket `vars`, key `vars.<extId>.<token>.last_read_at`
//     with extId = our extension id): written only by us on
//     POST /sessions/{id}/read. Read semantics live entirely on the
//     platform side; the agent neither stores nor interprets them.
//
// unread_count for a session = number of messages in its chain newer than
// the watermark. We do NOT keep message rows; the agent's projection gives
// us the newest timestamp, so the precise count is recomputed lazily via
// the existing per-session messages API only when a session is flagged
// unread (typically 0-2 sessions per list render). Presence of a fact newer
// than the watermark is enough to light the badge in the meantime.
package sessionstate

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/bus"
	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/extension"
	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/protocol"
	"github.com/zergx-platform/zergx/internal/upstream"
)

const (
	// ExtID is our extension id on the bus. Read watermarks are keyed under
	// vars.<ExtID>.<sessionToken>.<name>.
	ExtID = "platform"

	// StateBucket carries agent-published message facts. DELIBERATELY not
	// the SDK lease bucket (abc-session-state in agent/lease.ts), which the
	// session runner uses for mutual exclusion with a 30s TTL — sharing it
	// would let a fact overwrite the run lease and stall turns.
	StateBucket = "abc-session-meta"

	// readVar is the read-watermark variable name (session scope).
	readVar = "last_read_at"
)

// Fact is one session's newest-message projection published by the agent,
// plus a lazily-refined unread count (refined via the agent messages API —
// the KV fact itself carries no count by design).
type Fact struct {
	LastMessageAt      string `json:"last_message_at"`
	LastMessagePreview string `json:"last_message_preview"`
	LastMessageRole    string `json:"last_message_role"`

	unread      int  // 0 or 1 until refined
	unreadExact bool // true after RefineUnread computed the real count
}

// Session summarizes one session for the chat list. Facts come from the
// agent projection; ReadAt from our watermark store. UnreadCount > 0 means
// the badge should render; it is a lower bound refined lazily by the caller
// via the per-session messages API.
type Session struct {
	Name             string
	LastMessageAt    string
	LastMessagePrev  string
	LastMessageRole  string
	ReadAt           string // zero value = never opened
	UnreadCount      int
	UnreadCalculated bool // false → UnreadCount is a lower bound (0/1)
}

// Store watches the bus and maintains in-memory views of both flows.
type Store struct {
	bus      bus.Bus
	ext      *extension.Extension
	client   *upstream.Client // agent; nil in tests → no lazy refinement
	mu       sync.RWMutex
	facts    map[string]Fact   // sessionToken -> agent fact
	waterMk  map[string]string // sessionToken -> read watermark
	stopCh   chan struct{}
	stopOnce sync.Once
}

// New attaches to the bus and starts both watchers. All failures inside the
// watchers degrade gracefully: missing facts/watermarks simply render as an
// already-read list.
func New(b bus.Bus, agent *upstream.Client) *Store {
	s := &Store{
		bus:     b,
		client:  agent,
		facts:   map[string]Fact{},
		waterMk: map[string]string{},
		stopCh:  make(chan struct{}),
	}
	go s.watchFacts()
	go s.loadWatermarks()
	return s
}

// SetExt registers the extension handle used for watermark writes.
func (s *Store) SetExt(ext *extension.Extension) { s.ext = ext }

// Close stops the watchers.
func (s *Store) Close() { s.stopOnce.Do(func() { close(s.stopCh) }) }

// watchFacts consumes the agent's message-fact projection.
func (s *Store) watchFacts() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	events, stop, err := s.bus.KvWatch(ctx, StateBucket, ">")
	if err != nil {
		// Bus topology not reconciled yet (first agent boot); facts degrade
		// to absent and appear as soon as any message lands.
		slog.Warn("session-state watch failed", "err", err)
		return
	}
	defer stop()
	for {
		select {
		case <-s.stopCh:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Deleted || ev.Value == "" {
				continue
			}
			tok := tokenOfKey(ev.Key)
			if tok == "" {
				continue
			}
			var f Fact
			if json.Unmarshal([]byte(ev.Value), &f) != nil {
				continue
			}
			s.mu.Lock()
			// A new message invalidates the prior refined count.
			f.unread = 0
			f.unreadExact = false
			s.facts[tok] = f
			s.mu.Unlock()
		}
	}
}

// tokenOfKey strips nothing: keys inside the bucket ARE the session token
// (`sha256(sessionName)[:22]`, irreversible by design — protocol.SessionToken).
// Facts are therefore indexed by token; Snapshot() converts session names to
// tokens on lookup.
func tokenOfKey(key string) string { return key }

// loadWatermarks pulls the persisted read watermarks once at startup, then
// keeps the in-memory map updated via watch. Watermarks are authoritative
// for read state; facts only light the badge.
func (s *Store) loadWatermarks() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	// NATS KV watch filters are relative to the bucket subject
	// ($KV.<bucket>.<pattern>), so the pattern must NOT repeat the bucket.
	events, stop, err := s.bus.KvWatch(ctx, protocol.VarsBucket, ExtID+".*."+readVar)
	if err != nil {
		slog.Warn("read-watermark watch failed", "err", err)
		return
	}
	defer stop()
	for {
		select {
		case <-s.stopCh:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			sid := watermarkKeyToSession(ev.Key)
			if sid == "" {
				continue
			}
			s.mu.Lock()
			if ev.Deleted || ev.Value == "" {
				delete(s.waterMk, sid)
			} else {
				s.waterMk[sid] = ev.Value
			}
			s.mu.Unlock()
		}
	}
}

// watermarkKeyToSession extracts the session token from a vars watch key.
// Watch delivers keys relative to the bucket: `<extId>.<token>.<name>`.
func watermarkKeyToSession(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) != 3 || parts[0] != ExtID || parts[2] != readVar {
		return ""
	}
	return parts[1]
}

// MarkRead persists a read watermark for one session.
func (s *Store) MarkRead(ctx context.Context, sid string) error {
	if s.ext == nil {
		return errNotReady
	}
	if err := s.ext.SetSessionVariable(ctx, sid, readVar, nowStr()); err != nil {
		return err
	}
	// A fresh watermark invalidates any previously-refined exact count.
	s.mu.Lock()
	tok := protocol.SessionToken(sid)
	if f, ok := s.facts[tok]; ok {
		f.unread = 0
		f.unreadExact = false
		s.facts[tok] = f
	}
	s.mu.Unlock()
	return nil
}

var errNotReady = &notReadyError{}

type notReadyError struct{}

func (*notReadyError) Error() string { return "bus extension not attached" }

// Snapshot folds facts and watermarks into per-session summaries for the
// given session names. Sessions without a fact are omitted from the result.
func (s *Store) Snapshot(names []string) map[string]Session {
	out := map[string]Session{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, name := range names {
		tok := protocol.SessionToken(name)
		f, ok := s.facts[tok]
		if !ok {
			continue
		}
		ss := Session{
			Name:            name,
			LastMessageAt:   f.LastMessageAt,
			LastMessagePrev: f.LastMessagePreview,
			LastMessageRole: f.LastMessageRole,
		}
		wm, hasWM := s.waterMk[tok]
		ss.ReadAt = wm
		unread := 0
		exact := f.unreadExact
		if !hasWM {
			// Never opened: everything after the (absent) watermark is
			// unread. Lower bound 1 until refined.
			unread = 1
		} else if wm != "" && f.LastMessageAt > wm {
			// Watermark timestamp ordering: both are `YYYY-MM-DD HH:MM:SS`
			// strings, lexicographic order == chronological order.
			unread = 1
		}
		if f.unreadExact {
			unread = f.unread
		}
		ss.UnreadCount = unread
		ss.UnreadCalculated = exact
		// Every session with a fact is returned — preview/timestamp are
		// read-independent chat-list facts. unread_count is gated separately
		// by mergeSessionState (only emitted when > 0).
		out[name] = ss
	}
	return out
}

func nowStr() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// RefineUnread recomputes exact unread counts for sessions whose badge is
// lit, using the per-session messages API (limit=1 against the agent would
// give only the newest message; the count needs the full chain, so we use
// the messages endpoint with a generous limit and count rows newer than the
// watermark). Typically 0-2 sessions are flagged; the cost is bounded.
func (s *Store) RefineUnread(ctx context.Context, names []string) {
	for _, name := range names {
		s.mu.RLock()
		ss, ok := s.unsafeSnapshotOne(name)
		s.mu.RUnlock()
		if !ok || ss.UnreadCount == 0 || ss.UnreadCalculated {
			continue
		}
		count, ok := s.countUnread(ctx, name, ss.ReadAt)
		if !ok {
			continue
		}
		s.mu.Lock()
		tok := protocol.SessionToken(name)
		if f, ok := s.facts[tok]; ok {
			f.unread = count
			f.unreadExact = true
			s.facts[tok] = f
		}
		s.mu.Unlock()
	}
}

// unsafeSnapshotOne is Snapshot for one name; caller holds at least RLock.
func (s *Store) unsafeSnapshotOne(name string) (Session, bool) {
	tok := protocol.SessionToken(name)
	f, ok := s.facts[tok]
	if !ok {
		return Session{}, false
	}
	ss := Session{Name: name, LastMessageAt: f.LastMessageAt,
		LastMessagePrev: f.LastMessagePreview, LastMessageRole: f.LastMessageRole}
	wm, hasWM := s.waterMk[tok]
	ss.ReadAt = wm
	if !hasWM {
		ss.UnreadCount = 1
	} else if wm != "" && f.LastMessageAt > wm {
		ss.UnreadCount = 1
	}
	return ss, true
}

// countUnread asks the agent for the session chain and counts rows newer
// than the watermark. best-effort: on any error the lower bound stands.
func (s *Store) countUnread(ctx context.Context, sid, watermark string) (int, bool) {
	if s.client == nil {
		return 0, false
	}
	// The full chain may be long; unread can never exceed what arrived after
	// the watermark, and chat lists only need a display count. Cap at 200.
	var res struct {
		Messages []struct {
			CreatedAt string `json:"created_at"`
		} `json:"messages"`
	}
	q := url.Values{}
	q.Set("limit", "200")
	if err := s.client.JSON(ctx, http.MethodGet,
		"/api/v1/sessions/"+url.PathEscape(sid)+"/messages",
		nil, q, &res); err != nil {
		return 0, false
	}
	n := 0
	for _, m := range res.Messages {
		if m.CreatedAt > watermark {
			n++
		}
	}
	return n, true
}

// NewTestStore builds a Store pre-seeded with facts and watermarks, without
// attaching a bus or agent client. For aggregate-level tests only.
func NewTestStore(
	facts map[string]Fact,
	watermarks map[string]string,
) *Store {
	s := &Store{
		facts:   map[string]Fact{},
		waterMk: map[string]string{},
		stopCh:  make(chan struct{}),
	}
	for k, v := range facts {
		s.facts[k] = v
	}
	for k, v := range watermarks {
		s.waterMk[k] = v
	}
	return s
}
