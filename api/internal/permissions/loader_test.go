package permissions

import (
	"testing"

	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/runfivedev/runfive/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.UserServerRole{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func createRole(t *testing.T, db *gorm.DB, name, globalPerms, serverPerms string) models.Role {
	t.Helper()
	role := models.Role{
		Name:        name,
		Color:       "#000000",
		GlobalPerms: globalPerms,
		ServerPerms: serverPerms,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("failed to create role %q: %v", name, err)
	}
	return role
}

func createUser(t *testing.T, db *gorm.DB, username string, isOwner bool, globalRoleID *uint) models.User {
	t.Helper()
	user := models.User{
		Username:     username,
		IsOwner:      isOwner,
		GlobalRoleID: globalRoleID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user %q: %v", username, err)
	}
	return user
}

func assignServerRole(t *testing.T, db *gorm.DB, userID uint, serverID string, roleID uint) {
	t.Helper()
	assignment := models.UserServerRole{
		UserID:   userID,
		ServerID: serverID,
		RoleID:   roleID,
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("failed to assign server role: %v", err)
	}
}

func TestLoadForUser_Owner(t *testing.T) {
	db := setupTestDB(t)
	owner := createUser(t, db, "owner", true, nil)

	perms, err := LoadForUser(db, &owner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.IsOwner {
		t.Fatal("expected IsOwner to be true")
	}

	// Owner should have full access to all global resources
	for resource, actions := range GlobalResourceActions {
		for _, action := range actions {
			if !perms.Global.Has(resource, action) {
				t.Errorf("owner missing global permission %s.%s", resource, action)
			}
		}
	}

	// Owner should have no server-specific entries (handled at middleware level)
	if len(perms.Servers) != 0 {
		t.Errorf("expected empty server perms for owner, got %d", len(perms.Servers))
	}

	if perms.GlobalRole != nil {
		t.Error("expected nil GlobalRole for owner")
	}
}

func TestLoadForUser_NoRoles(t *testing.T) {
	db := setupTestDB(t)
	user := createUser(t, db, "noroles", false, nil)

	perms, err := LoadForUser(db, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if perms.IsOwner {
		t.Fatal("expected IsOwner to be false")
	}

	if len(perms.Global) != 0 {
		t.Errorf("expected empty global perms, got %d resources", len(perms.Global))
	}
	if len(perms.Servers) != 0 {
		t.Errorf("expected empty server perms, got %d servers", len(perms.Servers))
	}
	if perms.GlobalRole != nil {
		t.Error("expected nil GlobalRole")
	}
}

func TestLoadForUser_GlobalRoleOnly(t *testing.T) {
	db := setupTestDB(t)
	role := createRole(t, db, "Moderator",
		`{"users":{"read":true},"roles":{"read":true}}`,
		`{"dashboard":{"read":true}}`,
	)
	user := createUser(t, db, "mod", false, &role.ID)

	perms, err := LoadForUser(db, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Global perms from role
	if !perms.Global.Has("users", "read") {
		t.Error("expected users.read to be granted")
	}
	if !perms.Global.Has("roles", "read") {
		t.Error("expected roles.read to be granted")
	}
	if perms.Global.Has("users", "delete") {
		t.Error("users.delete should NOT be granted")
	}
	if perms.Global.Has("settings", "read") {
		t.Error("settings.read should NOT be granted")
	}

	// No server roles assigned
	if len(perms.Servers) != 0 {
		t.Errorf("expected no server perms, got %d", len(perms.Servers))
	}

	// GlobalRole metadata
	if perms.GlobalRole == nil {
		t.Fatal("expected GlobalRole to be set")
	}
	if perms.GlobalRole.Name != "Moderator" {
		t.Errorf("expected role name Moderator, got %q", perms.GlobalRole.Name)
	}
}

func TestLoadForUser_ServerRolesOnly(t *testing.T) {
	db := setupTestDB(t)
	adminRole := createRole(t, db, "Admin",
		`{}`,
		`{"players":{"read":true,"kick":true},"bans":{"read":true,"create":true}}`,
	)
	viewerRole := createRole(t, db, "Viewer",
		`{}`,
		`{"players":{"read":true},"bans":{"read":true}}`,
	)
	user := createUser(t, db, "staffer", false, nil)

	assignServerRole(t, db, user.ID, "server-alpha", adminRole.ID)
	assignServerRole(t, db, user.ID, "server-beta", viewerRole.ID)

	perms, err := LoadForUser(db, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No global perms (no global role)
	if len(perms.Global) != 0 {
		t.Errorf("expected empty global perms, got %d", len(perms.Global))
	}

	// Server Alpha: Admin role
	alphaPerms, ok := perms.Servers["server-alpha"]
	if !ok {
		t.Fatal("expected server-alpha in server perms")
	}
	if !alphaPerms.Has("players", "read") {
		t.Error("server-alpha: expected players.read")
	}
	if !alphaPerms.Has("players", "kick") {
		t.Error("server-alpha: expected players.kick")
	}
	if !alphaPerms.Has("bans", "create") {
		t.Error("server-alpha: expected bans.create")
	}

	// Server Beta: Viewer role
	betaPerms, ok := perms.Servers["server-beta"]
	if !ok {
		t.Fatal("expected server-beta in server perms")
	}
	if !betaPerms.Has("players", "read") {
		t.Error("server-beta: expected players.read")
	}
	if betaPerms.Has("players", "kick") {
		t.Error("server-beta: players.kick should NOT be granted (viewer role)")
	}
	if betaPerms.Has("bans", "create") {
		t.Error("server-beta: bans.create should NOT be granted (viewer role)")
	}

	// Server role metadata
	alphaMeta, ok := perms.ServerRoles["server-alpha"]
	if !ok {
		t.Fatal("expected server-alpha role metadata")
	}
	if alphaMeta.Name != "Admin" {
		t.Errorf("expected role name Admin, got %q", alphaMeta.Name)
	}

	betaMeta, ok := perms.ServerRoles["server-beta"]
	if !ok {
		t.Fatal("expected server-beta role metadata")
	}
	if betaMeta.Name != "Viewer" {
		t.Errorf("expected role name Viewer, got %q", betaMeta.Name)
	}
}

func TestLoadForUser_GlobalAndServerRoles(t *testing.T) {
	db := setupTestDB(t)

	globalRole := createRole(t, db, "PanelMod",
		`{"users":{"read":true}}`,
		`{}`,
	)
	serverRole := createRole(t, db, "ServerAdmin",
		`{}`,
		`{"players":{"read":true,"update":true,"kick":true},"console":{"read":true,"execute":true}}`,
	)
	user := createUser(t, db, "hybrid", false, &globalRole.ID)
	assignServerRole(t, db, user.ID, "prod-1", serverRole.ID)

	perms, err := LoadForUser(db, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Global from global role
	if !perms.Global.Has("users", "read") {
		t.Error("expected global users.read")
	}
	if perms.Global.Has("users", "delete") {
		t.Error("global users.delete should NOT be granted")
	}

	// Server from server role
	prod1, ok := perms.Servers["prod-1"]
	if !ok {
		t.Fatal("expected prod-1 in server perms")
	}
	if !prod1.Has("players", "kick") {
		t.Error("prod-1: expected players.kick")
	}
	if !prod1.Has("console", "execute") {
		t.Error("prod-1: expected console.execute")
	}
	if prod1.Has("bans", "read") {
		t.Error("prod-1: bans.read should NOT be granted")
	}

	// No cross-contamination: server role perms don't appear in global
	if perms.Global.Has("players", "read") {
		t.Error("server-level players.read should NOT appear in global perms")
	}
}

func TestLoadForUser_MalformedGlobalPermsJSON(t *testing.T) {
	db := setupTestDB(t)
	role := createRole(t, db, "Broken", `{invalid json`, `{}`)
	user := createUser(t, db, "user", false, &role.ID)

	_, err := LoadForUser(db, &user)
	if err == nil {
		t.Fatal("expected error for malformed global perms JSON")
	}
}

func TestLoadForUser_MalformedServerPermsJSON(t *testing.T) {
	db := setupTestDB(t)
	role := createRole(t, db, "Broken", `{}`, `{invalid json`)

	user := createUser(t, db, "user", false, nil)
	assignServerRole(t, db, user.ID, "srv", role.ID)

	_, err := LoadForUser(db, &user)
	if err == nil {
		t.Fatal("expected error for malformed server perms JSON")
	}
}

func TestLoadForUser_DeletedGlobalRole(t *testing.T) {
	db := setupTestDB(t)
	role := createRole(t, db, "Temp", `{"users":{"read":true}}`, `{}`)
	user := createUser(t, db, "orphan", false, &role.ID)

	// Hard-delete the role
	db.Unscoped().Delete(&role)

	_, err := LoadForUser(db, &user)
	if err == nil {
		t.Fatal("expected error when global role is deleted")
	}
}

func TestLoadForUser_ServerIdIsNotForeignKey(t *testing.T) {
	// ServerID is a free-form string (TOML directory name), not a DB FK.
	// Any string value should work, including special characters.
	db := setupTestDB(t)
	role := createRole(t, db, "Test", `{}`, `{"players":{"read":true}}`)
	user := createUser(t, db, "user", false, nil)

	assignServerRole(t, db, user.ID, "my-server_v2.prod", role.ID)

	perms, err := LoadForUser(db, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := perms.Servers["my-server_v2.prod"]; !ok {
		t.Fatal("expected server ID with special characters to work")
	}
}

func TestLoadForUser_UniqueConstraintPerServer(t *testing.T) {
	// A user should only have one role per server (enforced by unique index).
	db := setupTestDB(t)
	role1 := createRole(t, db, "Role1", `{}`, `{"players":{"read":true}}`)
	role2 := createRole(t, db, "Role2", `{}`, `{"players":{"read":true,"kick":true}}`)
	user := createUser(t, db, "user", false, nil)

	assignServerRole(t, db, user.ID, "srv", role1.ID)

	// Second assignment to same server should fail
	dup := models.UserServerRole{UserID: user.ID, ServerID: "srv", RoleID: role2.ID}
	err := db.Create(&dup).Error
	if err == nil {
		t.Fatal("expected unique constraint violation for duplicate user+server assignment")
	}
}
