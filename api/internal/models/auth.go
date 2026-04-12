// Package models provides request and response DTOs for authentication endpoints.
package models

// LoginRequest is the body for POST /v1/auth/login.
type LoginRequest struct {
	// Username of the account
	Username string `json:"username" validate:"required,min=3,max=32"`
	// Plaintext password
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest is the body for POST /v1/auth/register (master account setup).
type RegisterRequest struct {
	// Desired username for the master account
	Username string `json:"username" validate:"required,min=3,max=32"`
	// Plaintext password (min 8 chars)
	Password string `json:"password" validate:"required,min=8"`
	// Code is the formatted setup token ("xxxx-xxxx") printed to the
	// server console at first startup. Required to bootstrap the owner.
	Code string `json:"code" validate:"required,len=9"`
}

// RoleInfo contains display metadata for a role in API responses.
type RoleInfo struct {
	// ID is the role database ID.
	ID uint `json:"id"`
	// Name is the role display name.
	Name string `json:"name"`
	// Color is the hex color for UI badges.
	Color string `json:"color"`
}

// ServerPermissionEntry is the per-server permission block in the /me response.
type ServerPermissionEntry struct {
	// Role is the role metadata for this server assignment.
	Role RoleInfo `json:"role"`
	// Permissions is the resolved server-scoped permission map.
	Permissions map[string]map[string]bool `json:"permissions"`
}

// MeResponse is returned by GET /v1/auth/me.
// Includes the user's resolved global and per-server permissions so the
// frontend can conditionally render UI elements without extra API calls.
type MeResponse struct {
	// User database ID
	ID uint `json:"id"`
	// Username
	Username string `json:"username"`
	// IsOwner indicates whether this user is the owner (master account).
	IsOwner bool `json:"isOwner"`
	// Providers contains linked authentication providers.
	Providers ProviderInfo `json:"providers"`
	// GlobalRole is the user's panel-wide role, nil if none assigned.
	GlobalRole *RoleInfo `json:"globalRole"`
	// GlobalPermissions is the resolved panel-wide permission map.
	GlobalPermissions map[string]map[string]bool `json:"globalPermissions"`
	// ServerPermissions maps server ID to role + permissions for that server.
	ServerPermissions map[string]ServerPermissionEntry `json:"serverPermissions"`
}

// ProviderInfo contains optional linked OAuth provider details.
type ProviderInfo struct {
	// Cfx is the Cfx.re account info, nil if not linked.
	Cfx *CfxInfo `json:"cfx"`
	// Discord is the Discord account info, nil if not linked.
	Discord *DiscordInfo `json:"discord"`
}

// CfxInfo contains Cfx.re (Discourse) account details.
type CfxInfo struct {
	// ID is the Discourse user ID on forum.cfx.re.
	ID int `json:"id"`
	// Username is the Discourse username.
	Username string `json:"username"`
	// AvatarURL is the avatar URL template.
	AvatarURL string `json:"avatarUrl"`
}

// DiscordInfo contains Discord account details (planned).
type DiscordInfo struct {
	// ID is the Discord user ID (snowflake).
	ID string `json:"id"`
	// Username is the Discord username.
	Username string `json:"username"`
	// Avatar is the Discord avatar hash.
	Avatar string `json:"avatar"`
}

// SessionResponse is a single entry in the GET /v1/auth/sessions list.
type SessionResponse struct {
	// ID is the session database ID.
	ID uint `json:"id"`
	// UserAgent is the client User-Agent.
	UserAgent string `json:"userAgent"`
	// CreatedAt is when the session was created.
	CreatedAt string `json:"createdAt"`
	// LastSeenAt is when the session was last active.
	LastSeenAt string `json:"lastSeenAt"`
	// IsCurrent indicates whether this is the session making the current request.
	IsCurrent bool `json:"isCurrent"`
}

// SetupStatusResponse is returned by GET /v1/auth/setup-status.
type SetupStatusResponse struct {
	// NeedsSetup is true if no users exist and the master account needs to be created.
	NeedsSetup bool `json:"needsSetup"`
}

// UserListItem is a single entry in GET /v1/users.
type UserListItem struct {
	ID          uint         `json:"id"`
	Username    string       `json:"username"`
	IsOwner     bool         `json:"isOwner"`
	HasPassword bool         `json:"hasPassword"`
	Providers   ProviderInfo `json:"providers"`
	SuspendedAt *string      `json:"suspendedAt"`
	CreatedAt   string       `json:"createdAt"`
	// GlobalRole is the user's panel-wide role, nil if none assigned.
	GlobalRole *RoleInfo `json:"globalRole"`
	// ServerRoleCount is the number of server-specific role assignments.
	ServerRoleCount int `json:"serverRoleCount"`
}

// UserServerRoleEntry is a single server role assignment in the user detail response.
type UserServerRoleEntry struct {
	// ServerID is the server directory name.
	ServerID string `json:"serverId"`
	// Role is the assigned role metadata.
	Role RoleInfo `json:"role"`
}

// RoleListItem is a single entry in GET /v1/roles.
type RoleListItem struct {
	// ID is the role database ID.
	ID uint `json:"id"`
	// Name is the role display name.
	Name string `json:"name"`
	// Description is an optional explanation.
	Description string `json:"description"`
	// Color is the hex color for UI badges.
	Color string `json:"color"`
	// GlobalPerms is the raw JSON of panel-wide permissions.
	GlobalPerms map[string]map[string]bool `json:"globalPerms"`
	// ServerPerms is the raw JSON of server resource permissions.
	ServerPerms map[string]map[string]bool `json:"serverPerms"`
	// IsSystem is true for seeded roles that cannot be deleted.
	IsSystem bool `json:"isSystem"`
	// Position is the display order.
	Position int `json:"position"`
	// AssignedUsers is the count of users assigned to this role (global + server).
	AssignedUsers int `json:"assignedUsers"`
}

// CreateRoleRequest is the body for POST /v1/roles.
type CreateRoleRequest struct {
	Name        string                     `json:"name" validate:"required,min=1,max=64"`
	Description string                     `json:"description" validate:"max=255"`
	Color       string                     `json:"color" validate:"required,len=7"`
	GlobalPerms map[string]map[string]bool `json:"globalPerms"`
	ServerPerms map[string]map[string]bool `json:"serverPerms"`
}

// UpdateRoleRequest is the body for PUT /v1/roles/:id.
type UpdateRoleRequest struct {
	Name        *string                     `json:"name" validate:"omitempty,min=1,max=64"`
	Description *string                     `json:"description" validate:"omitempty,max=255"`
	Color       *string                     `json:"color" validate:"omitempty,len=7"`
	GlobalPerms *map[string]map[string]bool `json:"globalPerms"`
	ServerPerms *map[string]map[string]bool `json:"serverPerms"`
	Position    *int                        `json:"position"`
}

// SetGlobalRoleRequest is the body for PUT /v1/users/:id/global-role.
type SetGlobalRoleRequest struct {
	// RoleID is the role to assign, or null to remove.
	RoleID *uint `json:"roleId"`
}

// SetServerRoleRequest is the body for PUT /v1/users/:id/server-roles/:serverId.
type SetServerRoleRequest struct {
	// RoleID is the role to assign on this server.
	RoleID uint `json:"roleId" validate:"required"`
}

// ResourceSchemaDTO describes the actions available for a single resource.
type ResourceSchemaDTO struct {
	// CRUD lists which CRUD actions apply to this resource.
	CRUD []string `json:"crud"`
	// Sub lists resource-specific actions beyond CRUD (e.g. kick, warn, execute).
	Sub []string `json:"sub,omitempty"`
}

// PermissionSchemaResponse is returned by GET /v1/permissions/schema.
type PermissionSchemaResponse struct {
	// Global maps global resource name to its action schema.
	Global map[string]ResourceSchemaDTO `json:"global"`
	// Server maps server resource name to its action schema.
	Server map[string]ResourceSchemaDTO `json:"server"`
}
