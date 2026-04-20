package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/runfivedev/runfive/internal/fxserver"
	"github.com/runfivedev/runfive/internal/launcher"
	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/serverfs"
)

// --- fakes ---------------------------------------------------------------

type fakeRegistry struct {
	servers map[string]models.ManagedServer
}

func (f *fakeRegistry) List() ([]models.ManagedServer, error) {
	out := make([]models.ManagedServer, 0, len(f.servers))
	for id := range f.servers {
		out = append(out, f.servers[id])
	}
	return out, nil
}

func (f *fakeRegistry) Get(id string) (models.ManagedServer, bool) {
	s, ok := f.servers[id]
	return s, ok
}

func (f *fakeRegistry) Create(string, string, string, int, int) (models.ManagedServer, error) {
	return models.ManagedServer{}, errors.New("not implemented")
}

func (f *fakeRegistry) Update(string, *serverfs.UpdatePatch) (models.ManagedServer, error) {
	return models.ManagedServer{}, errors.New("not implemented")
}

func (f *fakeRegistry) Delete(string, bool) error { return errors.New("not implemented") }
func (f *fakeRegistry) Reload() error             { return nil }

type fakeArtifacts struct{}

func (fakeArtifacts) Install(context.Context, string) (models.InstalledArtifact, error) {
	return models.InstalledArtifact{}, errors.New("not implemented")
}

type fakeLauncher struct {
	status    models.ServerProcessStatus
	statusErr error
}

func (f *fakeLauncher) Start(string) (models.ServerProcessStatus, error) {
	return models.ServerProcessStatus{}, errors.New("not implemented")
}

func (f *fakeLauncher) Stop(string) (models.ServerProcessStatus, error) {
	return models.ServerProcessStatus{}, errors.New("not implemented")
}

func (f *fakeLauncher) Status(string) (models.ServerProcessStatus, error) {
	return f.status, f.statusErr
}

func (f *fakeLauncher) Tail(string, int) ([]models.ServerLogLine, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLauncher) Subscribe(string) (*launcher.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLauncher) SendCommand(string, string) error { return errors.New("not implemented") }
func (f *fakeLauncher) IsRunning(string) bool            { return false }

type fakeRuntimeClient struct {
	players   []fxserver.Player
	err       error
	lastPort  int
	callCount int
}

func (f *fakeRuntimeClient) FetchPlayers(_ context.Context, port int) ([]fxserver.Player, error) {
	f.lastPort = port
	f.callCount++
	return f.players, f.err
}

// --- helpers -------------------------------------------------------------

type playersDeps struct {
	registry *fakeRegistry
	launcher *fakeLauncher
	runtime  *fakeRuntimeClient
}

func newPlayersApp(deps playersDeps) *fiber.App {
	app := fiber.New()
	handler := NewServerHandler(deps.registry, fakeArtifacts{}, deps.launcher, deps.runtime)
	app.Get("/v1/servers/:serverId/players", handler.Players)
	return app
}

func doPlayersRequest(t *testing.T, app *fiber.App, serverID string) (status int, body []byte) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"/v1/servers/"+serverID+"/players", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode, body
}

func decodePlayers(t *testing.T, body []byte) []fxserver.Player {
	t.Helper()
	var players []fxserver.Player
	if err := json.Unmarshal(body, &players); err != nil {
		t.Fatalf("decode body %q: %v", string(body), err)
	}
	return players
}

// --- tests ---------------------------------------------------------------

func TestPlayers_UnknownServer(t *testing.T) {
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{}},
		launcher: &fakeLauncher{},
		runtime:  &fakeRuntimeClient{},
	}
	app := newPlayersApp(deps)

	status, _ := doPlayersRequest(t, app, "ghost")
	if status != http.StatusNotFound {
		t.Fatalf("unknown server: want 404, got %d", status)
	}
	if deps.runtime.callCount != 0 {
		t.Errorf("runtime client must not be called for unknown server")
	}
}

func TestPlayers_ServerStopped(t *testing.T) {
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 30120},
		}},
		launcher: &fakeLauncher{status: models.ServerProcessStatus{Status: models.ServerStatusStopped}},
		runtime:  &fakeRuntimeClient{},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("stopped server: want 200, got %d", status)
	}
	if got := decodePlayers(t, body); len(got) != 0 {
		t.Errorf("stopped server: want empty list, got %+v", got)
	}
	if deps.runtime.callCount != 0 {
		t.Errorf("runtime client must not be called for stopped server")
	}
}

func TestPlayers_LauncherStatusError(t *testing.T) {
	// Registry unaware of launcher failure — e.g. a race between reload and
	// the launcher's internal state. Endpoint stays calm and reports zero
	// players rather than surfacing a transient internal error.
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 30120},
		}},
		launcher: &fakeLauncher{statusErr: errors.New("transient")},
		runtime:  &fakeRuntimeClient{},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("launcher error: want 200, got %d", status)
	}
	if got := decodePlayers(t, body); len(got) != 0 {
		t.Errorf("launcher error: want empty list, got %+v", got)
	}
	if deps.runtime.callCount != 0 {
		t.Errorf("runtime client must not be called on launcher error")
	}
}

func TestPlayers_RunningButNoPort(t *testing.T) {
	// Port == 0 means the registry entry is downgraded / unresolved. We
	// cannot hit loopback without a port; return an empty list rather than
	// fabricate a target.
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 0},
		}},
		launcher: &fakeLauncher{status: models.ServerProcessStatus{Status: models.ServerStatusRunning}},
		runtime:  &fakeRuntimeClient{},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("no port: want 200, got %d", status)
	}
	if got := decodePlayers(t, body); len(got) != 0 {
		t.Errorf("no port: want empty list, got %+v", got)
	}
	if deps.runtime.callCount != 0 {
		t.Errorf("runtime client must not be called when port is 0")
	}
}

func TestPlayers_RunningRuntimeError(t *testing.T) {
	// fxserver booted but HTTP listener not up yet / transient refused —
	// must not flap the UI into an error state while polling at 5 s.
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 30120},
		}},
		launcher: &fakeLauncher{status: models.ServerProcessStatus{Status: models.ServerStatusRunning}},
		runtime:  &fakeRuntimeClient{err: errors.New("connection refused")},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("runtime error: want 200, got %d", status)
	}
	if got := decodePlayers(t, body); len(got) != 0 {
		t.Errorf("runtime error: want empty list, got %+v", got)
	}
	if deps.runtime.callCount != 1 {
		t.Errorf("runtime client should have been called once, got %d", deps.runtime.callCount)
	}
}

func TestPlayers_RunningWithPlayers(t *testing.T) {
	want := []fxserver.Player{
		{ID: 1, Name: "Alice", Ping: 42, License: "abc", Discord: "111"},
		{ID: 2, Name: "Bob", Ping: 88, License: "def"},
	}
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 30120},
		}},
		launcher: &fakeLauncher{status: models.ServerProcessStatus{Status: models.ServerStatusRunning}},
		runtime:  &fakeRuntimeClient{players: want},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("want 200, got %d", status)
	}
	got := decodePlayers(t, body)
	if len(got) != len(want) {
		t.Fatalf("player count: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("player[%d]: want %+v, got %+v", i, want[i], got[i])
		}
	}
	if deps.runtime.lastPort != 30120 {
		t.Errorf("runtime called with port %d, want 30120", deps.runtime.lastPort)
	}
}

func TestPlayers_RunningEmpty(t *testing.T) {
	// fxserver up, zero clients connected. Endpoint must return an empty
	// JSON array — not null — so the frontend's length read is safe.
	deps := playersDeps{
		registry: &fakeRegistry{servers: map[string]models.ManagedServer{
			"alpha": {ID: "alpha", Port: 30120},
		}},
		launcher: &fakeLauncher{status: models.ServerProcessStatus{Status: models.ServerStatusRunning}},
		runtime:  &fakeRuntimeClient{players: []fxserver.Player{}},
	}
	app := newPlayersApp(deps)

	status, body := doPlayersRequest(t, app, "alpha")
	if status != http.StatusOK {
		t.Fatalf("want 200, got %d", status)
	}
	if string(body) != "[]" {
		t.Errorf("empty list must serialize as []; got %q", string(body))
	}
}
