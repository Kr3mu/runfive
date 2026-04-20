package fxserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout caps runtime queries short enough that a tick hitch trips
// fast, but long enough to survive the occasional GC pause on the fxserver
// side. Handlers still wrap this in a request-scoped context so Fiber can
// cancel the in-flight HTTP call when the client goes away.
const defaultTimeout = 2 * time.Second

// RuntimeClient fetches runtime data from a locally-running fxserver HTTP
// endpoint. The panel and every managed server share a host, so every query
// targets loopback — never the advertised address, which may be a DNS name
// that does not resolve the same way from inside the panel.
//
// Named to distinguish from scraper.go's Client, which talks to the cfx.re
// artifact index at build/install time and has nothing to do with a running
// server.
type RuntimeClient struct {
	http *http.Client
}

// NewRuntimeClient builds a RuntimeClient with the default 2s HTTP timeout.
func NewRuntimeClient() *RuntimeClient {
	return &RuntimeClient{http: &http.Client{Timeout: defaultTimeout}}
}

// rawPlayer mirrors the on-the-wire /players.json element. Kept unexported
// because the prefixed-string identifier layout is a detail of the fxserver
// protocol; callers get the parsed Player type instead.
type rawPlayer struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Ping        int      `json:"ping"`
	Endpoint    string   `json:"endpoint"`
	Identifiers []string `json:"identifiers"`
}

// Player is the typed, identifier-parsed player record returned to callers
// and serialized straight out of the API handler. Fields that depend on
// identifiers the server did not collect are empty strings rather than
// errors — a player without a linked Discord account is a normal state.
type Player struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Ping    int    `json:"ping"`
	License string `json:"license,omitempty"`
	Discord string `json:"discord,omitempty"`
}

// FetchPlayers GETs http://127.0.0.1:<port>/players.json and returns the
// parsed list. On connection refused, timeout, non-200 response, or
// malformed JSON the caller receives an error; the HTTP handler converts
// those failures into an empty list so the dashboard can keep polling
// without error toasts during boot windows.
func (c *RuntimeClient) FetchPlayers(ctx context.Context, port int) ([]Player, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/players.json", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fxserver returned status %d", resp.StatusCode)
	}

	var raw []rawPlayer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode players.json: %w", err)
	}

	players := make([]Player, 0, len(raw))
	for _, r := range raw {
		players = append(players, Player{
			ID:      r.ID,
			Name:    r.Name,
			Ping:    r.Ping,
			License: findIdentifier(r.Identifiers, "license"),
			Discord: findIdentifier(r.Identifiers, "discord"),
		})
	}
	return players, nil
}

// findIdentifier returns the value of the first identifier with the given
// prefix (without the prefix), or "" if none matches. fxserver only ever
// assigns a single identifier per provider per player, so returning the
// first hit matches the real-world shape.
func findIdentifier(identifiers []string, prefix string) string {
	needle := prefix + ":"
	for _, id := range identifiers {
		if strings.HasPrefix(id, needle) {
			return strings.TrimPrefix(id, needle)
		}
	}
	return ""
}
