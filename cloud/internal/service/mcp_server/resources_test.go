package mcp_server

import "testing"

func TestResourcesList_Returns6URIs(t *testing.T) {
	svc := NewResourceService()
	result := svc.ListResources()

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	resources, ok := resp["resources"].([]map[string]any)
	if !ok {
		t.Fatal("expected resources array")
	}
	if len(resources) != 6 {
		t.Fatalf("expected 6 resources, got %d", len(resources))
	}

	expectedURIs := []string{
		"dipole://audit-log-severities",
		"dipole://audit-log-categories",
		"dipole://auth-event-types",
		"dipole://auth-event-severities",
		"dipole://workload-event-types",
		"dipole://workload-event-severities",
	}

	for i, expected := range expectedURIs {
		uri, _ := resources[i]["uri"].(string)
		if uri != expected {
			t.Errorf("resource[%d] URI = %q, want %q", i, uri, expected)
		}
	}
}

func TestResourcesRead_AuditLogSeverities(t *testing.T) {
	svc := NewResourceService()
	result, err := svc.ReadResource("dipole://audit-log-severities")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	contents, ok := resp["contents"].([]map[string]any)
	if !ok || len(contents) == 0 {
		t.Fatal("expected contents array")
	}
	if contents[0]["uri"] != "dipole://audit-log-severities" {
		t.Error("expected URI in content")
	}
}

func TestResourcesRead_AuditLogCategories(t *testing.T) {
	svc := NewResourceService()
	result, err := svc.ReadResource("dipole://audit-log-categories")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := result.(map[string]any)
	contents := resp["contents"].([]map[string]any)
	text := contents[0]["text"]

	categories, ok := text.([]string)
	if !ok {
		t.Fatal("expected string array for categories")
	}
	if len(categories) != 24 {
		t.Fatalf("expected 24 categories, got %d", len(categories))
	}
}

func TestResourcesRead_WorkloadEventTypes(t *testing.T) {
	svc := NewResourceService()
	result, err := svc.ReadResource("dipole://workload-event-types")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := result.(map[string]any)
	contents := resp["contents"].([]map[string]any)
	text := contents[0]["text"]

	types, ok := text.([]string)
	if !ok {
		t.Fatal("expected string array for workload event types")
	}
	if len(types) != 9 {
		t.Fatalf("expected 9 protocols, got %d", len(types))
	}
}

func TestResourcesRead_NotFound(t *testing.T) {
	svc := NewResourceService()
	_, err := svc.ReadResource("dipole://nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown resource URI")
	}
}
