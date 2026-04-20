package fxserver

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// newTestServer starts an httptest.Server on a loopback port and returns
// only that port. FetchPlayers hardcodes 127.0.0.1 so we rely on httptest's
// default binding there; the server itself is torn down via t.Cleanup.
func newTestServer(t *testing.T, handler http.HandlerFunc) int {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}
	return port
}

func TestFetchPlayers_Success(t *testing.T) {
	port := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/players.json" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"name":"Alice","ping":42,"endpoint":"1.2.3.4:5","identifiers":["license:abc","discord:111","steam:222"]},
			{"id":2,"name":"Bob","ping":88,"endpoint":"6.7.8.9:10","identifiers":["license:def"]}
		]`))
	})

	players, err := NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err != nil {
		t.Fatalf("FetchPlayers: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("want 2 players, got %d", len(players))
	}

	alice := players[0]
	if alice.ID != 1 || alice.Name != "Alice" || alice.Ping != 42 {
		t.Errorf("alice scalar fields wrong: %+v", alice)
	}
	if alice.License != "abc" {
		t.Errorf("alice license: want abc, got %q", alice.License)
	}
	if alice.Discord != "111" {
		t.Errorf("alice discord: want 111, got %q", alice.Discord)
	}

	bob := players[1]
	if bob.License != "def" {
		t.Errorf("bob license: want def, got %q", bob.License)
	}
	if bob.Discord != "" {
		t.Errorf("bob discord: want empty, got %q", bob.Discord)
	}
}

func TestFetchPlayers_EmptyList(t *testing.T) {
	port := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	players, err := NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err != nil {
		t.Fatalf("FetchPlayers: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("want empty slice, got %d entries", len(players))
	}
}

func TestFetchPlayers_MissingIdentifiers(t *testing.T) {
	port := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// No identifiers at all — license and discord should both end up "".
		_, _ = w.Write([]byte(`[{"id":1,"name":"Ghost","ping":0,"identifiers":[]}]`))
	})

	players, err := NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err != nil {
		t.Fatalf("FetchPlayers: %v", err)
	}
	if len(players) != 1 || players[0].License != "" || players[0].Discord != "" {
		t.Errorf("missing identifiers should produce empty strings, got %+v", players)
	}
}

func TestFetchPlayers_NonOKStatus(t *testing.T) {
	port := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	_, err := NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err == nil {
		t.Fatal("want error on 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code, got %q", err.Error())
	}
}

func TestFetchPlayers_MalformedJSON(t *testing.T) {
	port := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})

	_, err := NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err == nil {
		t.Fatal("want decode error, got nil")
	}
}

func TestFetchPlayers_ConnectionRefused(t *testing.T) {
	// Bind then immediately close to get a guaranteed-free port.
	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	_, err = NewRuntimeClient().FetchPlayers(context.Background(), port)
	if err == nil {
		t.Fatal("want connect error, got nil")
	}
}
