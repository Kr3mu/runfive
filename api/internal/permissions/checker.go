package permissions

import "encoding/json"

// PermissionMap is the parsed form of a permission JSON blob.
// Outer key is the resource name, inner key is the action.
type PermissionMap map[string]map[string]bool

// Parse deserializes a JSON permission string into a PermissionMap.
// Returns an empty map on empty string or "{}".
func Parse(jsonStr string) (PermissionMap, error) {
	if jsonStr == "" || jsonStr == "{}" {
		return PermissionMap{}, nil
	}
	var pm PermissionMap
	if err := json.Unmarshal([]byte(jsonStr), &pm); err != nil {
		return nil, err
	}
	return pm, nil
}

// Has returns true if the given resource+action is granted.
// Missing keys are treated as denied (deny by default).
func (pm PermissionMap) Has(resource, action string) bool {
	if pm == nil {
		return false
	}
	actions, ok := pm[resource]
	if !ok {
		return false
	}
	return actions[action]
}

// IsSubsetOf returns true if every granted permission in pm is also granted in other.
// Used for privilege escalation checks: a non-owner can only create/edit roles
// whose permissions are a subset of their own.
func (pm PermissionMap) IsSubsetOf(other PermissionMap) bool {
	for resource, actions := range pm {
		for action, granted := range actions {
			if granted && !other.Has(resource, action) {
				return false
			}
		}
	}
	return true
}

// FullAccessMap returns a PermissionMap where every resource+action is true.
// Used to build the owner's effective permissions.
func FullAccessMap(resourceActions map[string][]string) PermissionMap {
	pm := make(PermissionMap, len(resourceActions))
	for resource, actions := range resourceActions {
		actionMap := make(map[string]bool, len(actions))
		for _, action := range actions {
			actionMap[action] = true
		}
		pm[resource] = actionMap
	}
	return pm
}

// ToJSON serializes a PermissionMap to a JSON string.
func (pm PermissionMap) ToJSON() (string, error) {
	if pm == nil {
		return "{}", nil
	}
	b, err := json.Marshal(pm)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
