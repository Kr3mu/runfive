// Package permissions defines the canonical permission registry, checker,
// and per-request loader for the RBAC system.
//
// Resources and actions are defined as constants here (single source of truth).
// To add a new server resource, add one constant and append to AllServerResources.
// Each resource has optional CRUD actions and optional sub-actions (e.g. kick, warn).
package permissions

// CRUD actions.
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

// CRUDActions is the canonical set of CRUD actions.
var CRUDActions = []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}

// Sub-actions (resource-specific, not CRUD).
const (
	ActionKick    = "kick"
	ActionWarn    = "warn"
	ActionExecute = "execute"
)

// ResourceSchema defines the available actions for a resource,
// split into CRUD actions and resource-specific sub-actions.
type ResourceSchema struct {
	// CRUD lists which of the four CRUD actions apply (nil entries omitted).
	CRUD []string `json:"crud"`
	// Sub lists resource-specific actions beyond CRUD (e.g. kick, warn, execute).
	Sub []string `json:"sub,omitempty"`
}

// AllActions returns the combined list of CRUD + Sub actions.
func (rs ResourceSchema) AllActions() []string {
	all := make([]string, 0, len(rs.CRUD)+len(rs.Sub))
	all = append(all, rs.CRUD...)
	all = append(all, rs.Sub...)
	return all
}

// Global resources (panel-wide, not scoped to a server).
const (
	GlobalUsers    = "users"
	GlobalRoles    = "roles"
	GlobalServers  = "servers"
	GlobalSettings = "settings"
)

// AllGlobalResources is the canonical list of global resources.
var AllGlobalResources = []string{GlobalUsers, GlobalRoles, GlobalServers, GlobalSettings}

// GlobalResourceSchema maps each global resource to its action schema.
//
// GlobalServers.update grants a platform-wide operator the ability to edit any
// server's config without holding a per-server role. Day-to-day config edits
// are expected to flow through the per-server ServerSettings resource instead
// so blast radius stays bounded.
var GlobalResourceSchema = map[string]ResourceSchema{
	GlobalUsers:    {CRUD: []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}},
	GlobalRoles:    {CRUD: []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}},
	GlobalServers:  {CRUD: []string{ActionCreate, ActionUpdate, ActionDelete}},
	GlobalSettings: {CRUD: []string{ActionRead, ActionUpdate}},
}

// GlobalResourceActions is a flat map for backward compatibility with middleware.
// Maps resource -> all actions (CRUD + sub).
var GlobalResourceActions = flattenSchemas(GlobalResourceSchema)

// Server resources (scoped to a specific server).
const (
	ServerDashboard = "dashboard"
	ServerPlayers   = "players"
	ServerConsole   = "console"
	ServerBans      = "bans"
	ServerSettings  = "settings"
)

// AllServerResources is the canonical list of server resources.
var AllServerResources = []string{ServerDashboard, ServerPlayers, ServerConsole, ServerBans, ServerSettings}

// ServerResourceSchema maps each server resource to its action schema.
//
// ServerSettings covers the per-server server.toml. update lets an operator
// edit their own server's config; delete is deliberately absent — deleting a
// server is a platform-level action and lives on GlobalServers.delete.
var ServerResourceSchema = map[string]ResourceSchema{
	ServerDashboard: {CRUD: []string{ActionRead}},
	ServerPlayers:   {CRUD: []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}, Sub: []string{ActionKick, ActionWarn}},
	ServerConsole:   {CRUD: []string{ActionRead}, Sub: []string{ActionExecute}},
	ServerBans:      {CRUD: []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}},
	ServerSettings:  {CRUD: []string{ActionRead, ActionUpdate}},
}

// ServerResourceActions is a flat map for backward compatibility with middleware.
var ServerResourceActions = flattenSchemas(ServerResourceSchema)

// flattenSchemas converts ResourceSchema maps to flat action lists.
func flattenSchemas(schemas map[string]ResourceSchema) map[string][]string {
	flat := make(map[string][]string, len(schemas))
	for resource, schema := range schemas {
		flat[resource] = schema.AllActions()
	}
	return flat
}
