package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPatches_WithAnnotation_HasInitAndSidecar(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test-tenant",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller.dipole:80",
	}}

	patches := inj.buildPatches(map[string]string{"dipolehq.com/agent-inject": "enabled"}, 1)

	if len(patches) != 2 {
		t.Fatalf("expected 2 patches (init + sidecar), got %d", len(patches))
	}
	if patches[0].Path != "/spec/initContainers/-" {
		t.Errorf("patch[0] path = %q", patches[0].Path)
	}
	if patches[1].Path != "/spec/containers/-" {
		t.Errorf("patch[1] path = %q", patches[1].Path)
	}
}

func TestBuildPatches_ExplicitMode_NoInit_HasHTTPProxy(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller:80",
	}}

	patches := inj.buildPatches(map[string]string{
		"dipolehq.com/agent-inject":  "enabled",
		"dipolehq.com/steering-mode": "explicit",
	}, 2)

	for _, p := range patches {
		if p.Path == "/spec/initContainers/-" {
			t.Error("explicit mode should NOT have init container")
		}
	}

	httpProxyCount := 0
	for _, p := range patches {
		if val, ok := p.Value.(map[string]interface{}); ok {
			if val["name"] == "HTTP_PROXY" {
				httpProxyCount++
			}
		}
	}
	if httpProxyCount != 2 {
		t.Errorf("expected 2 HTTP_PROXY patches, got %d", httpProxyCount)
	}
}

func TestBuildPatches_WithResourceSetID(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller:80",
	}}

	patches := inj.buildPatches(map[string]string{
		"dipolehq.com/agent-inject":    "enabled",
		"dipolehq.com/resource-set-id": "rs-uuid-123",
	}, 1)

	sidecarJSON, _ := json.Marshal(patches[1].Value)
	if !strings.Contains(string(sidecarJSON), "DIPOLE_RESOURCE_SET_ID") {
		t.Error("expected DIPOLE_RESOURCE_SET_ID in sidecar env")
	}
	if !strings.Contains(string(sidecarJSON), "rs-uuid-123") {
		t.Error("expected rs-uuid-123 value")
	}
}

func TestBuildPatches_CustomEnvAnnotation(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller:80",
	}}

	patches := inj.buildPatches(map[string]string{
		"dipolehq.com/agent-inject":           "enabled",
		"dipolehq.com/agent-proxy-env-MY_VAR": "my-value",
	}, 1)

	sidecarJSON, _ := json.Marshal(patches[1].Value)
	if !strings.Contains(string(sidecarJSON), "MY_VAR") {
		t.Error("expected MY_VAR in sidecar env")
	}
	if !strings.Contains(string(sidecarJSON), "my-value") {
		t.Error("expected my-value")
	}
}
