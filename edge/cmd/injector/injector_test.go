package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPatches_NoExistingInitContainers_CreatesArray(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test-tenant",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller.dipole:80",
	}}

	patches := inj.buildPatches(map[string]string{"dipolehq.com/agent-inject": "enabled"}, 1, false, false)

	if patches[0].Path != "/spec/initContainers" {
		t.Errorf("patch[0] path = %q, want /spec/initContainers (create array)", patches[0].Path)
	}

	hasVolumePatch := false
	hasSidecar := false
	hasSSLCertFile := false
	for _, p := range patches {
		if p.Path == "/spec/volumes" {
			hasVolumePatch = true
		}
		if p.Path == "/spec/containers/-" {
			hasSidecar = true
		}
		if strings.Contains(p.Path, "/env/-") {
			if val, ok := p.Value.(map[string]interface{}); ok {
				if val["name"] == "SSL_CERT_FILE" {
					hasSSLCertFile = true
				}
			}
		}
	}
	if !hasVolumePatch {
		t.Error("expected /spec/volumes patch for CA volume")
	}
	if !hasSidecar {
		t.Error("expected sidecar container patch")
	}
	if !hasSSLCertFile {
		t.Error("expected SSL_CERT_FILE env injection into app container")
	}
}

func TestBuildPatches_HasExistingInitContainers_Appends(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test-tenant",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller.dipole:80",
	}}

	patches := inj.buildPatches(map[string]string{"dipolehq.com/agent-inject": "enabled"}, 1, true, false)

	if patches[0].Path != "/spec/initContainers/-" {
		t.Errorf("patch[0] path = %q, want /spec/initContainers/- (append)", patches[0].Path)
	}
}

func TestBuildPatches_HasExistingVolumes_AppendsVolume(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test-tenant",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller.dipole:80",
	}}

	patches := inj.buildPatches(map[string]string{"dipolehq.com/agent-inject": "enabled"}, 1, false, true)

	hasVolumeAppend := false
	for _, p := range patches {
		if p.Path == "/spec/volumes/-" {
			hasVolumeAppend = true
		}
	}
	if !hasVolumeAppend {
		t.Error("expected /spec/volumes/- patch (append to existing volumes)")
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
	}, 2, false, false)

	for _, p := range patches {
		if strings.Contains(p.Path, "initContainers") {
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
	}, 1, false, false)

	found := false
	for _, p := range patches {
		b, _ := json.Marshal(p.Value)
		if strings.Contains(string(b), "DIPOLE_RESOURCE_SET_ID") && strings.Contains(string(b), "rs-uuid-123") {
			found = true
		}
	}
	if !found {
		t.Error("expected DIPOLE_RESOURCE_SET_ID=rs-uuid-123 in sidecar env")
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
	}, 1, false, false)

	found := false
	for _, p := range patches {
		b, _ := json.Marshal(p.Value)
		if strings.Contains(string(b), "MY_VAR") && strings.Contains(string(b), "my-value") {
			found = true
		}
	}
	if !found {
		t.Error("expected MY_VAR=my-value in sidecar env")
	}
}

func TestBuildPatches_SidecarHasCAVolumeMount(t *testing.T) {
	inj := &AgentInjector{config: &InjectorConfig{
		TenantID:      "test",
		ProxyImage:    "dipole/proxy:latest",
		InitImage:     "dipole/init:latest",
		ControllerURL: "http://controller:80",
	}}

	patches := inj.buildPatches(map[string]string{"dipolehq.com/agent-inject": "enabled"}, 1, false, false)

	for _, p := range patches {
		if p.Path == "/spec/containers/-" {
			b, _ := json.Marshal(p.Value)
			if !strings.Contains(string(b), "dipole-ca") {
				t.Error("sidecar container should have dipole-ca volume mount")
			}
			if !strings.Contains(string(b), "/dipole/ca") {
				t.Error("sidecar container should mount at /dipole/ca")
			}
			return
		}
	}
	t.Error("sidecar container patch not found")
}
