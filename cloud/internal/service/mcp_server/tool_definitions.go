/*
tool_definitions.go contains the static MCP tool schemas returned by tools/list.
*/
package mcp_server

var toolDefinitions = []map[string]any{
	{
		"name":        "get_audit_logs",
		"description": "Retrieve audit logs with filtering and pagination. Returns admin activity logs. By default returns the last 30 days sorted by newest first.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"page":            map[string]any{"type": "integer", "default": 1, "description": "Page number for pagination"},
				"perPage":         map[string]any{"type": "integer", "default": 100, "description": "Results per page. Maximum 100."},
				"orderBy":         map[string]any{"type": "string", "default": "CreatedAt", "enum": []string{"CreatedAt", "Category", "ActorDisplayName", "Activity", "Target", "OutcomeResult", "Severity"}},
				"descending":      map[string]any{"type": "boolean", "default": true},
				"search":          map[string]any{"type": "string", "default": "", "description": "Search text to filter"},
				"startDate":       map[string]any{"type": "string", "description": "Start of date range ISO 8601. Requires endDate."},
				"endDate":         map[string]any{"type": "string", "description": "End of date range ISO 8601. Clamped to now if future."},
				"spanLastMinutes": map[string]any{"type": "integer", "default": 0, "description": "Return logs from last N minutes. Priority over spanLastDays."},
				"spanLastDays":    map[string]any{"type": "integer", "default": 30, "description": "Return logs from last N days."},
				"category":        map[string]any{"type": "string", "description": "Filter by category. See audit_log_categories resource."},
				"severity":        map[string]any{"type": "string", "enum": []string{"Info", "Warn", "Alert"}},
			},
		},
	},
	{
		"name":        "get_auth_events",
		"description": "Retrieve access authorization events with filtering. Returns policy evaluation outcomes. Default last 24 hours sorted by newest.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"page":            map[string]any{"type": "integer", "default": 1},
				"perPage":         map[string]any{"type": "integer", "default": 100},
				"orderBy":         map[string]any{"type": "string", "default": "Timestamp", "enum": []string{"Timestamp", "ClientIp", "ContextId", "ClientWorkload", "ServerWorkload", "Severity"}},
				"descending":      map[string]any{"type": "boolean", "default": true},
				"search":          map[string]any{"type": "string", "default": ""},
				"startDate":       map[string]any{"type": "string"},
				"endDate":         map[string]any{"type": "string"},
				"spanLastMinutes": map[string]any{"type": "integer", "default": 0},
				"spanLastHours":   map[string]any{"type": "integer", "default": 24},
				"severity":        map[string]any{"type": "string", "enum": []string{"Error", "Alert", "Warn", "Info"}},
				"eventType":       map[string]any{"type": "string", "enum": []string{"Request", "Authorization", "Credential"}, "description": "Filter by event type"},
			},
		},
	},
	{
		"name":        "get_workload_events",
		"description": "Retrieve workload events with filtering. Returns protocol-level request/response records. Default last 24 hours.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"page":            map[string]any{"type": "integer", "default": 1},
				"perPage":         map[string]any{"type": "integer", "default": 100},
				"orderBy":         map[string]any{"type": "string", "default": "Timestamp", "enum": []string{"Timestamp", "ConnectionId", "EventType", "ClientWorkload", "ServerWorkload", "Severity"}},
				"descending":      map[string]any{"type": "boolean", "default": true},
				"search":          map[string]any{"type": "string", "default": ""},
				"startDate":       map[string]any{"type": "string"},
				"endDate":         map[string]any{"type": "string"},
				"spanLastMinutes": map[string]any{"type": "integer", "default": 0},
				"spanLastHours":   map[string]any{"type": "integer", "default": 24},
				"severity":        map[string]any{"type": "string", "enum": []string{"Error", "Alert", "Warn", "Info"}},
				"appProtocol":     map[string]any{"type": "string", "enum": []string{"Redshift", "HTTP", "MySQL", "Postgres", "Redis", "Snowflake", "TCP", "OracleDatabase", "MCP"}},
				"sourceWorkload":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Filter by client workload IDs (UUIDs)"},
				"targetWorkload":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Filter by server workload IDs (UUIDs)"},
			},
		},
	},
}
