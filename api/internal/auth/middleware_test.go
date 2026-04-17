package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/permissions"
)

// injectPerms returns middleware that sets resolved permissions in Fiber locals.
func injectPerms(rp *permissions.ResolvedPermissions) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals(localsPermsKey, rp)
		return c.Next()
	}
}

// injectUser returns middleware that sets an authenticated user in Fiber locals.
func injectUser(user *models.User) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals(localsUserKey, user)
		return c.Next()
	}
}

func okHandler(c fiber.Ctx) error {
	return c.SendString("OK")
}

func doRequest(t *testing.T, app *fiber.App, path string) int {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// --- RequireGlobalPerm Tests ---

func TestRequireGlobalPerm_OwnerBypasses(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{IsOwner: true}))
	app.Get("/test", RequireGlobalPerm("users", "delete"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 200 {
		t.Fatalf("expected 200 for owner, got %d", status)
	}
}

func TestRequireGlobalPerm_GrantedPermission(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 200 {
		t.Fatalf("expected 200 for granted permission, got %d", status)
	}
}

func TestRequireGlobalPerm_DeniedPermission(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "delete"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 for denied permission, got %d", status)
	}
}

func TestRequireGlobalPerm_MissingResource(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("roles", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 for missing resource, got %d", status)
	}
}

func TestRequireGlobalPerm_NoPermissionsLoaded(t *testing.T) {
	app := fiber.New()
	// No injectPerms — Locals("permissions") is nil
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 when no permissions loaded, got %d", status)
	}
}

func TestRequireGlobalPerm_EmptyPermissions(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 for empty permissions, got %d", status)
	}
}

func TestRequireGlobalPerm_ExplicitFalseIsDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": false}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 for explicit false, got %d", status)
	}
}

// --- RequireServerPerm Tests ---

func TestRequireServerPerm_OwnerBypasses(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{IsOwner: true}))
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "read"), okHandler)

	status := doRequest(t, app, "/servers/my-srv/players")
	if status != 200 {
		t.Fatalf("expected 200 for owner, got %d", status)
	}
}

func TestRequireServerPerm_GrantedPermission(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"players": {"read": true, "kick": true}},
		},
	}))
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "kick"), okHandler)

	status := doRequest(t, app, "/servers/prod-1/players")
	if status != 200 {
		t.Fatalf("expected 200 for granted server permission, got %d", status)
	}
}

func TestRequireServerPerm_WrongServer(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"players": {"read": true}},
		},
	}))
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "read"), okHandler)

	// User has access to prod-1, but requests prod-2
	status := doRequest(t, app, "/servers/prod-2/players")
	if status != 403 {
		t.Fatalf("expected 403 for wrong server, got %d", status)
	}
}

func TestRequireServerPerm_DeniedAction(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"players": {"read": true}},
		},
	}))
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "delete"), okHandler)

	status := doRequest(t, app, "/servers/prod-1/players")
	if status != 403 {
		t.Fatalf("expected 403 for denied action, got %d", status)
	}
}

func TestRequireServerPerm_MissingServerIdParam(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"players": {"read": true}},
		},
	}))
	// Route without :serverId param
	app.Get("/test", RequireServerPerm("players", "read"), okHandler)

	status := doRequest(t, app, "/test")
	if status != 400 {
		t.Fatalf("expected 400 for missing serverId param, got %d", status)
	}
}

func TestRequireServerPerm_NoPermissionsLoaded(t *testing.T) {
	app := fiber.New()
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "read"), okHandler)

	status := doRequest(t, app, "/servers/prod-1/players")
	if status != 403 {
		t.Fatalf("expected 403 when no permissions loaded, got %d", status)
	}
}

func TestRequireServerPerm_ServerIdTampering(t *testing.T) {
	// User has access to server-a with players.kick,
	// but tries to access server-b which they have no role on.
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"server-a": {"players": {"read": true, "kick": true}},
		},
	}))
	app.Get("/servers/:serverId/players/kick", RequireServerPerm("players", "kick"), okHandler)

	// Legitimate request
	status := doRequest(t, app, "/servers/server-a/players/kick")
	if status != 200 {
		t.Fatalf("expected 200 for legitimate server, got %d", status)
	}

	// Tampered request
	status = doRequest(t, app, "/servers/server-b/players/kick")
	if status != 403 {
		t.Fatalf("expected 403 for tampered server ID, got %d", status)
	}
}

// --- GetPermissions Tests ---

func TestGetPermissions_ReturnsNilWhenNotSet(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		perms := GetPermissions(c)
		if perms != nil {
			return c.SendString("FAIL: expected nil")
		}
		return c.SendString("OK")
	})

	status := doRequest(t, app, "/test")
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
}

func TestGetPermissions_ReturnsPermsWhenSet(t *testing.T) {
	rp := &permissions.ResolvedPermissions{IsOwner: true}
	app := fiber.New()
	app.Use(injectPerms(rp))
	app.Get("/test", func(c fiber.Ctx) error {
		perms := GetPermissions(c)
		if perms == nil {
			return c.SendString("FAIL: expected non-nil")
		}
		if !perms.IsOwner {
			return c.SendString("FAIL: expected IsOwner")
		}
		return c.SendString("OK")
	})

	status := doRequest(t, app, "/test")
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
}

// --- RequireServerOrGlobalPerm Tests ---

func TestRequireServerOrGlobalPerm_OwnerBypasses(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{IsOwner: true}))
	app.Put("/servers/:serverId", RequireServerOrGlobalPerm("settings", "update", "servers", "update"), okHandler)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/servers/my-srv", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for owner, got %d", resp.StatusCode)
	}
}

func TestRequireServerOrGlobalPerm_GlobalPermissionAdmits(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"servers": {"update": true}},
	}))
	app.Put("/servers/:serverId", RequireServerOrGlobalPerm("settings", "update", "servers", "update"), okHandler)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/servers/prod-1", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for global permission, got %d", resp.StatusCode)
	}
}

func TestRequireServerOrGlobalPerm_PerServerPermissionAdmits(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"settings": {"update": true}},
		},
	}))
	app.Put("/servers/:serverId", RequireServerOrGlobalPerm("settings", "update", "servers", "update"), okHandler)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/servers/prod-1", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for server-scoped permission, got %d", resp.StatusCode)
	}
}

func TestRequireServerOrGlobalPerm_WrongServerDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Servers: map[string]permissions.PermissionMap{
			"prod-1": {"settings": {"update": true}},
		},
	}))
	app.Put("/servers/:serverId", RequireServerOrGlobalPerm("settings", "update", "servers", "update"), okHandler)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/servers/prod-2", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for wrong server, got %d", resp.StatusCode)
	}
}

func TestRequireServerOrGlobalPerm_NoPermissionsDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global:  permissions.PermissionMap{},
		Servers: map[string]permissions.PermissionMap{},
	}))
	app.Put("/servers/:serverId", RequireServerOrGlobalPerm("settings", "update", "servers", "update"), okHandler)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/servers/prod-1", http.NoBody)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 without either permission, got %d", resp.StatusCode)
	}
}

// --- RequireMaster Tests ---

func TestRequireMaster_OwnerAllowed(t *testing.T) {
	app := fiber.New()
	app.Use(injectUser(&models.User{IsOwner: true}))
	app.Get("/test", RequireMaster, okHandler)

	status := doRequest(t, app, "/test")
	if status != 200 {
		t.Fatalf("expected 200 for owner, got %d", status)
	}
}

func TestRequireMaster_NonOwnerDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectUser(&models.User{IsOwner: false}))
	app.Get("/test", RequireMaster, okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 for non-owner, got %d", status)
	}
}

func TestRequireMaster_NoUserDenied(t *testing.T) {
	app := fiber.New()
	app.Get("/test", RequireMaster, okHandler)

	status := doRequest(t, app, "/test")
	if status != 403 {
		t.Fatalf("expected 403 when no user set, got %d", status)
	}
}
