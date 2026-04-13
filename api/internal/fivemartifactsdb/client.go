// Package fivemartifactsdb queries the community-maintained jgscripts artifact
// database at https://artifacts.jgscripts.com for recommended builds and known
// broken artifact versions.
//
// Results are cached in memory with a short TTL so we do not hammer the upstream
// on every artifact list request.
package fivemartifactsdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultEndpoint is the canonical jsonv2 endpoint documented in the repo.
const DefaultEndpoint = "https://artifacts.jgscripts.com/jsonv2"

// DefaultCacheTTL is how long a successful fetch is considered fresh.
const DefaultCacheTTL = 5 * time.Minute

// BrokenArtifact is a single broken-build entry.
//
// The `Version` field may be a single build number ("8509") or a dash-separated
// range ("10268-10309"). Use DB.BrokenReasons to get a flat map.
type BrokenArtifact struct {
	/** Version number or range string as reported upstream. */
	Version string `json:"artifact"`
	/** Human-readable description of the issue. */
	Reason string `json:"reason"`
}

// DB is the parsed response from the jsonv2 endpoint.
type DB struct {
	/** Latest build number the community considers stable. */
	RecommendedArtifact string `json:"recommendedArtifact"`
	/** Direct Windows download URL for the recommended build (not used by us). */
	WindowsDownloadLink string `json:"windowsDownloadLink"`
	/** Direct Linux download URL for the recommended build (not used by us). */
	LinuxDownloadLink string `json:"linuxDownloadLink"`
	/** All known broken artifacts, as reported in the upstream db.json. */
	BrokenArtifacts []BrokenArtifact `json:"brokenArtifacts"`
}

// BrokenReasons expands range entries like "10268-10309" into a flat map of
// version → reason so callers can do a single lookup per artifact version.
//
// The expansion is bounded to 10000 entries per range to guard against malformed
// upstream data.
func (db *DB) BrokenReasons() map[string]string {
	reasons := make(map[string]string)
	if db == nil {
		return reasons
	}

	for _, entry := range db.BrokenArtifacts {
		value := strings.TrimSpace(entry.Version)
		if value == "" {
			continue
		}

		if !strings.Contains(value, "-") {
			reasons[value] = entry.Reason
			continue
		}

		start, end, ok := parseRange(value)
		if !ok {
			reasons[value] = entry.Reason
			continue
		}

		const maxRangeSize = 10000
		if end-start >= maxRangeSize {
			reasons[value] = entry.Reason
			continue
		}

		for v := start; v <= end; v++ {
			reasons[strconv.Itoa(v)] = entry.Reason
		}
	}

	return reasons
}

// Client fetches and caches DB results.
type Client struct {
	endpoint string
	ttl      time.Duration
	http     *http.Client

	mu      sync.Mutex
	cache   *DB
	fetched time.Time
}

// NewClient returns a Client that talks to DefaultEndpoint with DefaultCacheTTL.
func NewClient() *Client {
	return &Client{
		endpoint: DefaultEndpoint,
		ttl:      DefaultCacheTTL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchDB returns the parsed DB, using the in-memory cache when still fresh.
//
// A stale entry is returned if the network call fails, so a transient upstream
// outage does not take down the /v1/artifacts endpoint. The caller cannot tell
// fresh from stale — that is intentional, the data is advisory, not critical.
func (c *Client) FetchDB(ctx context.Context) (*DB, error) {
	c.mu.Lock()
	if c.cache != nil && time.Since(c.fetched) < c.ttl {
		cached := c.cache
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, http.NoBody)
	if err != nil {
		return c.fallback(fmt.Errorf("build request: %w", err))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return c.fallback(fmt.Errorf("fetch: %w", err))
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return c.fallback(fmt.Errorf("unexpected status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return c.fallback(fmt.Errorf("read body: %w", err))
	}

	var db DB
	if err := json.Unmarshal(body, &db); err != nil {
		return c.fallback(fmt.Errorf("decode body: %w", err))
	}

	c.mu.Lock()
	c.cache = &db
	c.fetched = time.Now()
	c.mu.Unlock()

	return &db, nil
}

// fallback returns the last successful cache entry when fresh fetch fails, or
// the original error if no cache exists.
func (c *Client) fallback(fetchErr error) (*DB, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache != nil {
		return c.cache, nil
	}
	return nil, errors.Join(errors.New("jgscripts db unavailable"), fetchErr)
}

func parseRange(value string) (start, end int, ok bool) {
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	if end < start {
		return 0, 0, false
	}
	return start, end, true
}
