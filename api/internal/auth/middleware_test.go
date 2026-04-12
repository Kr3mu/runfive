package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/Kr3mu/runfive/internal/permissions"
	"github.com/gofiber/fiber/v3"
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

func doRequest(t *testing.T, app *fiber.App, method, path string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	return resp
}

// --- RequireGlobalPerm Tests ---

func TestRequireGlobalPerm_OwnerBypasses(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{IsOwner: true}))
	app.Get("/test", RequireGlobalPerm("users", "delete"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for owner, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_GrantedPermission(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for granted permission, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_DeniedPermission(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "delete"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for denied permission, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_MissingResource(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": true}},
	}))
	app.Get("/test", RequireGlobalPerm("roles", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for missing resource, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_NoPermissionsLoaded(t *testing.T) {
	app := fiber.New()
	// No injectPerms — Locals("permissions") is nil
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 when no permissions loaded, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_EmptyPermissions(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for empty permissions, got %d", resp.StatusCode)
	}
}

func TestRequireGlobalPerm_ExplicitFalseIsDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{
		Global: permissions.PermissionMap{"users": {"read": false}},
	}))
	app.Get("/test", RequireGlobalPerm("users", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for explicit false, got %d", resp.StatusCode)
	}
}

// --- RequireServerPerm Tests ---

func TestRequireServerPerm_OwnerBypasses(t *testing.T) {
	app := fiber.New()
	app.Use(injectPerms(&permissions.ResolvedPermissions{IsOwner: true}))
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/servers/my-srv/players")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for owner, got %d", resp.StatusCode)
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

	resp := doRequest(t, app, "GET", "/servers/prod-1/players")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for granted server permission, got %d", resp.StatusCode)
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
	resp := doRequest(t, app, "GET", "/servers/prod-2/players")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for wrong server, got %d", resp.StatusCode)
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

	resp := doRequest(t, app, "GET", "/servers/prod-1/players")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for denied action, got %d", resp.StatusCode)
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

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing serverId param, got %d", resp.StatusCode)
	}
}

func TestRequireServerPerm_NoPermissionsLoaded(t *testing.T) {
	app := fiber.New()
	app.Get("/servers/:serverId/players", RequireServerPerm("players", "read"), okHandler)

	resp := doRequest(t, app, "GET", "/servers/prod-1/players")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 when no permissions loaded, got %d", resp.StatusCode)
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
	resp := doRequest(t, app, "GET", "/servers/server-a/players/kick")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for legitimate server, got %d", resp.StatusCode)
	}

	// Tampered request
	resp = doRequest(t, app, "GET", "/servers/server-b/players/kick")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for tampered server ID, got %d", resp.StatusCode)
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

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
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

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- RequireMaster Tests ---

func TestRequireMaster_OwnerAllowed(t *testing.T) {
	app := fiber.New()
	app.Use(injectUser(&models.User{IsOwner: true}))
	app.Get("/test", RequireMaster, okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for owner, got %d", resp.StatusCode)
	}
}

func TestRequireMaster_NonOwnerDenied(t *testing.T) {
	app := fiber.New()
	app.Use(injectUser(&models.User{IsOwner: false}))
	app.Get("/test", RequireMaster, okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for non-owner, got %d", resp.StatusCode)
	}
}

func TestRequireMaster_NoUserDenied(t *testing.T) {
	app := fiber.New()
	app.Get("/test", RequireMaster, okHandler)

	resp := doRequest(t, app, "GET", "/test")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 when no user set, got %d", resp.StatusCode)
	}
}
