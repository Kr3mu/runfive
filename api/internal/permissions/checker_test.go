package permissions

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectErr   bool
		expectEmpty bool
		checkKey    string
		checkAction string
		checkValue  bool
	}{
		{
			name:        "empty string returns empty map",
			input:       "",
			expectEmpty: true,
		},
		{
			name:        "empty JSON object returns empty map",
			input:       "{}",
			expectEmpty: true,
		},
		{
			name:        "valid JSON parses correctly",
			input:       `{"users":{"read":true,"delete":false}}`,
			checkKey:    "users",
			checkAction: "read",
			checkValue:  true,
		},
		{
			name:        "explicit false is preserved",
			input:       `{"users":{"read":false}}`,
			checkKey:    "users",
			checkAction: "read",
			checkValue:  false,
		},
		{
			name:      "malformed JSON returns error",
			input:     `{"users":`,
			expectErr: true,
		},
		{
			name:      "wrong type at top level returns error",
			input:     `"just a string"`,
			expectErr: true,
		},
		{
			name:      "wrong type for resource value returns error",
			input:     `{"users":"not an object"}`,
			expectErr: true,
		},
		{
			name:        "multiple resources parse correctly",
			input:       `{"users":{"read":true},"roles":{"create":true,"delete":false}}`,
			checkKey:    "roles",
			checkAction: "create",
			checkValue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := Parse(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectEmpty {
				if len(pm) != 0 {
					t.Fatalf("expected empty map, got %d entries", len(pm))
				}
				return
			}

			if tt.checkKey != "" {
				actions, ok := pm[tt.checkKey]
				if !ok {
					t.Fatalf("expected resource %q to exist", tt.checkKey)
				}
				got := actions[tt.checkAction]
				if got != tt.checkValue {
					t.Fatalf("expected %s.%s = %v, got %v", tt.checkKey, tt.checkAction, tt.checkValue, got)
				}
			}
		})
	}
}

func TestParse_RoundTrip(t *testing.T) {
	original := PermissionMap{
		"users": {"create": true, "read": true, "update": false, "delete": true},
		"roles": {"read": true},
	}

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	parsed, err := Parse(jsonStr)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for resource, actions := range original {
		for action, expected := range actions {
			got := parsed[resource][action]
			if got != expected {
				t.Errorf("round-trip mismatch: %s.%s expected %v, got %v", resource, action, expected, got)
			}
		}
	}
}

func TestHas(t *testing.T) {
	tests := []struct {
		name     string
		pm       PermissionMap
		resource string
		action   string
		expected bool
	}{
		{
			name:     "nil map returns false",
			pm:       nil,
			resource: "users",
			action:   "read",
			expected: false,
		},
		{
			name:     "empty map returns false",
			pm:       PermissionMap{},
			resource: "users",
			action:   "read",
			expected: false,
		},
		{
			name:     "resource missing returns false",
			pm:       PermissionMap{"roles": {"read": true}},
			resource: "users",
			action:   "read",
			expected: false,
		},
		{
			name:     "action missing from existing resource returns false",
			pm:       PermissionMap{"users": {"read": true}},
			resource: "users",
			action:   "delete",
			expected: false,
		},
		{
			name:     "explicit true returns true",
			pm:       PermissionMap{"users": {"read": true}},
			resource: "users",
			action:   "read",
			expected: true,
		},
		{
			name:     "explicit false returns false",
			pm:       PermissionMap{"users": {"read": false}},
			resource: "users",
			action:   "read",
			expected: false,
		},
		{
			name:     "case sensitive resource name",
			pm:       PermissionMap{"users": {"read": true}},
			resource: "Users",
			action:   "read",
			expected: false,
		},
		{
			name:     "case sensitive action name",
			pm:       PermissionMap{"users": {"read": true}},
			resource: "users",
			action:   "Read",
			expected: false,
		},
		{
			name:     "sub-action works the same as crud",
			pm:       PermissionMap{"players": {"kick": true}},
			resource: "players",
			action:   "kick",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pm.Has(tt.resource, tt.action)
			if got != tt.expected {
				t.Fatalf("Has(%q, %q) = %v, want %v", tt.resource, tt.action, got, tt.expected)
			}
		})
	}
}

func TestIsSubsetOf(t *testing.T) {
	tests := []struct {
		name     string
		pm       PermissionMap
		other    PermissionMap
		expected bool
	}{
		{
			name:     "nil is subset of anything",
			pm:       nil,
			other:    PermissionMap{"users": {"read": true}},
			expected: true,
		},
		{
			name:     "nil is subset of nil",
			pm:       nil,
			other:    nil,
			expected: true,
		},
		{
			name:     "empty is subset of anything",
			pm:       PermissionMap{},
			other:    PermissionMap{"users": {"read": true}},
			expected: true,
		},
		{
			name:     "empty is subset of empty",
			pm:       PermissionMap{},
			other:    PermissionMap{},
			expected: true,
		},
		{
			name:     "identical maps are subsets of each other",
			pm:       PermissionMap{"users": {"read": true, "create": true}},
			other:    PermissionMap{"users": {"read": true, "create": true}},
			expected: true,
		},
		{
			name:     "strict subset returns true",
			pm:       PermissionMap{"users": {"read": true}},
			other:    PermissionMap{"users": {"read": true, "create": true}},
			expected: true,
		},
		{
			name:     "superset returns false (privilege escalation)",
			pm:       PermissionMap{"users": {"read": true, "delete": true}},
			other:    PermissionMap{"users": {"read": true}},
			expected: false,
		},
		{
			name:     "disjoint resources returns false",
			pm:       PermissionMap{"roles": {"create": true}},
			other:    PermissionMap{"users": {"read": true}},
			expected: false,
		},
		{
			name:     "disjoint actions returns false",
			pm:       PermissionMap{"users": {"delete": true}},
			other:    PermissionMap{"users": {"read": true}},
			expected: false,
		},
		{
			name:     "explicit false in pm is not considered granted",
			pm:       PermissionMap{"users": {"read": false}},
			other:    PermissionMap{},
			expected: true,
		},
		{
			name:     "explicit false in pm with other having the perm",
			pm:       PermissionMap{"users": {"read": false, "delete": false}},
			other:    PermissionMap{"users": {"read": true}},
			expected: true,
		},
		{
			name:     "mixed true and false only checks granted ones",
			pm:       PermissionMap{"users": {"read": true, "delete": false}},
			other:    PermissionMap{"users": {"read": true}},
			expected: true,
		},
		{
			name:     "mixed: one granted perm missing from other",
			pm:       PermissionMap{"users": {"read": true, "delete": true}},
			other:    PermissionMap{"users": {"read": true}},
			expected: false,
		},
		{
			name:     "multiple resources partial overlap is false",
			pm:       PermissionMap{"users": {"read": true}, "roles": {"create": true}},
			other:    PermissionMap{"users": {"read": true}},
			expected: false,
		},
		{
			name:     "multiple resources full overlap is true",
			pm:       PermissionMap{"users": {"read": true}, "roles": {"create": true}},
			other:    PermissionMap{"users": {"read": true, "delete": true}, "roles": {"create": true, "read": true}},
			expected: true,
		},
		{
			name:     "sub-actions are checked the same way",
			pm:       PermissionMap{"players": {"kick": true, "warn": true}},
			other:    PermissionMap{"players": {"kick": true}},
			expected: false,
		},
		{
			name:     "sub-actions fully covered returns true",
			pm:       PermissionMap{"players": {"kick": true}},
			other:    PermissionMap{"players": {"kick": true, "warn": true, "read": true}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pm.IsSubsetOf(tt.other)
			if got != tt.expected {
				t.Fatalf("IsSubsetOf() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFullAccessMap(t *testing.T) {
	t.Run("generates all true for global resources", func(t *testing.T) {
		pm := FullAccessMap(GlobalResourceActions)

		for resource, actions := range GlobalResourceActions {
			for _, action := range actions {
				if !pm.Has(resource, action) {
					t.Errorf("FullAccessMap missing %s.%s", resource, action)
				}
			}
		}
	})

	t.Run("generates all true for server resources including sub-actions", func(t *testing.T) {
		pm := FullAccessMap(ServerResourceActions)

		for resource, actions := range ServerResourceActions {
			for _, action := range actions {
				if !pm.Has(resource, action) {
					t.Errorf("FullAccessMap missing %s.%s", resource, action)
				}
			}
		}

		// Explicitly verify sub-actions are included
		if !pm.Has("players", "kick") {
			t.Error("FullAccessMap missing players.kick sub-action")
		}
		if !pm.Has("players", "warn") {
			t.Error("FullAccessMap missing players.warn sub-action")
		}
		if !pm.Has("console", "execute") {
			t.Error("FullAccessMap missing console.execute sub-action")
		}
	})

	t.Run("empty input returns empty map", func(t *testing.T) {
		pm := FullAccessMap(map[string][]string{})
		if len(pm) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(pm))
		}
	})
}

func TestToJSON(t *testing.T) {
	t.Run("nil map returns empty object", func(t *testing.T) {
		var pm PermissionMap
		got, err := pm.ToJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "{}" {
			t.Fatalf("expected '{}', got %q", got)
		}
	})

	t.Run("empty map serializes to empty object", func(t *testing.T) {
		pm := PermissionMap{}
		got, err := pm.ToJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "{}" {
			t.Fatalf("expected '{}', got %q", got)
		}
	})

	t.Run("non-empty map serializes correctly", func(t *testing.T) {
		pm := PermissionMap{"users": {"read": true}}
		got, err := pm.ToJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Parse it back to verify
		parsed, err := Parse(got)
		if err != nil {
			t.Fatalf("failed to parse serialized JSON: %v", err)
		}
		if !parsed.Has("users", "read") {
			t.Fatal("parsed result missing users.read")
		}
	})
}

func TestResourceSchema_AllActions(t *testing.T) {
	t.Run("crud only", func(t *testing.T) {
		rs := ResourceSchema{CRUD: []string{"read", "create"}}
		got := rs.AllActions()
		if len(got) != 2 {
			t.Fatalf("expected 2 actions, got %d", len(got))
		}
	})

	t.Run("crud plus sub", func(t *testing.T) {
		rs := ResourceSchema{CRUD: []string{"read"}, Sub: []string{"kick", "warn"}}
		got := rs.AllActions()
		if len(got) != 3 {
			t.Fatalf("expected 3 actions, got %d", len(got))
		}
		// Verify order: CRUD first, then sub
		if got[0] != "read" || got[1] != "kick" || got[2] != "warn" {
			t.Fatalf("unexpected action order: %v", got)
		}
	})

	t.Run("nil sub returns only crud", func(t *testing.T) {
		rs := ResourceSchema{CRUD: []string{"read", "create"}, Sub: nil}
		got := rs.AllActions()
		if len(got) != 2 {
			t.Fatalf("expected 2 actions, got %d", len(got))
		}
	})
}

func TestFlattenSchemas(t *testing.T) {
	schemas := map[string]ResourceSchema{
		"players": {CRUD: []string{"read", "create"}, Sub: []string{"kick"}},
		"console": {CRUD: []string{"read"}, Sub: []string{"execute"}},
	}

	flat := flattenSchemas(schemas)

	if len(flat["players"]) != 3 {
		t.Fatalf("expected 3 actions for players, got %d", len(flat["players"]))
	}
	if len(flat["console"]) != 2 {
		t.Fatalf("expected 2 actions for console, got %d", len(flat["console"]))
	}
}

func TestRegistryConsistency(t *testing.T) {
	t.Run("GlobalResourceActions contains all GlobalResourceSchema entries", func(t *testing.T) {
		for resource, schema := range GlobalResourceSchema {
			flat, ok := GlobalResourceActions[resource]
			if !ok {
				t.Fatalf("GlobalResourceActions missing resource %q", resource)
			}
			expected := schema.AllActions()
			if len(flat) != len(expected) {
				t.Fatalf("resource %q: expected %d actions, got %d", resource, len(expected), len(flat))
			}
		}
	})

	t.Run("ServerResourceActions contains all ServerResourceSchema entries", func(t *testing.T) {
		for resource, schema := range ServerResourceSchema {
			flat, ok := ServerResourceActions[resource]
			if !ok {
				t.Fatalf("ServerResourceActions missing resource %q", resource)
			}
			expected := schema.AllActions()
			if len(flat) != len(expected) {
				t.Fatalf("resource %q: expected %d actions, got %d", resource, len(expected), len(flat))
			}
		}
	})

	t.Run("all global resources are listed in AllGlobalResources", func(t *testing.T) {
		for _, r := range AllGlobalResources {
			if _, ok := GlobalResourceSchema[r]; !ok {
				t.Fatalf("AllGlobalResources contains %q but GlobalResourceSchema does not", r)
			}
		}
		for r := range GlobalResourceSchema {
			found := false
			for _, listed := range AllGlobalResources {
				if listed == r {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("GlobalResourceSchema contains %q but AllGlobalResources does not", r)
			}
		}
	})

	t.Run("all server resources are listed in AllServerResources", func(t *testing.T) {
		for _, r := range AllServerResources {
			if _, ok := ServerResourceSchema[r]; !ok {
				t.Fatalf("AllServerResources contains %q but ServerResourceSchema does not", r)
			}
		}
		for r := range ServerResourceSchema {
			found := false
			for _, listed := range AllServerResources {
				if listed == r {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("ServerResourceSchema contains %q but AllServerResources does not", r)
			}
		}
	})
}
