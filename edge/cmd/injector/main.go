/*
Agent Injector — Kubernetes MutatingAdmissionWebhook that injects the Agent
Proxy sidecar into pods annotated with dipolehq.com/agent-inject: "enabled".
Adds init container (iptables, UID 26248) + sidecar (proxy, UID 65534).
Uses failurePolicy=Ignore so injector downtime never blocks pod creation.
*/
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := &InjectorConfig{
		TenantID:          os.Getenv("DIPOLE_TENANT_ID"),
		ProxyImage:        envOr("DIPOLE_PROXY_IMAGE", "dipole/proxy:latest"),
		InitImage:         envOr("DIPOLE_INIT_IMAGE", "dipole/init:latest"),
		ControllerURL:     envOr("DIPOLE_CONTROLLER_URL", "http://dipole-agent-controller.dipole-system.svc.cluster.local:80"),
		CloudGRPCEndpoint: os.Getenv("DIPOLE_CLOUD_GRPC_ENDPOINT"),
	}

	tlsCert, err := generateSelfSignedCert()
	if err != nil {
		logger.Error("generate tls cert", "error", err)
		os.Exit(1)
	}

	injector := &AgentInjector{config: cfg, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mutate", injector.handleMutate)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:      ":443",
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCert}},
	}

	go func() {
		logger.Info("injector webhook starting", "port", 443)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Error("webhook server error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	logger.Info("shutting down injector")
	server.Close()
}

type InjectorConfig struct {
	TenantID          string
	ProxyImage        string
	InitImage         string
	ControllerURL     string
	CloudGRPCEndpoint string
}

type AgentInjector struct {
	config *InjectorConfig
	logger *slog.Logger
}

type admissionReview struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Request    *admissionRequest  `json:"request,omitempty"`
	Response   *admissionResponse `json:"response,omitempty"`
}

type admissionRequest struct {
	UID    string          `json:"uid"`
	Object json.RawMessage `json:"object"`
}

type admissionResponse struct {
	UID       string  `json:"uid"`
	Allowed   bool    `json:"allowed"`
	PatchType *string `json:"patchType,omitempty"`
	Patch     []byte  `json:"patch,omitempty"`
}

type podObject struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Name string `json:"name"`
		} `json:"containers"`
	} `json:"spec"`
}

func (inj *AgentInjector) handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	var review admissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		http.Error(w, "decode review failed", http.StatusBadRequest)
		return
	}

	var pod podObject
	json.Unmarshal(review.Request.Object, &pod)

	resp := &admissionResponse{
		UID:     review.Request.UID,
		Allowed: true,
	}

	if pod.Metadata.Annotations["dipolehq.com/agent-inject"] == "enabled" {
		patches := inj.buildPatches(pod.Metadata.Annotations, len(pod.Spec.Containers))
		patchBytes, _ := json.Marshal(patches)
		patchType := "JSONPatch"
		resp.PatchType = &patchType
		resp.Patch = patchBytes
	}

	responseReview := admissionReview{
		APIVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
		Response:   resp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseReview)
}

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func (inj *AgentInjector) buildPatches(annotations map[string]string, containerCount int) []jsonPatch {
	var patches []jsonPatch

	steeringMode := annotations["dipolehq.com/steering-mode"]

	// Init container (skip for explicit steering)
	if steeringMode != "explicit" {
		patches = append(patches, jsonPatch{
			Op:   "add",
			Path: "/spec/initContainers/-",
			Value: map[string]interface{}{
				"name":  "dipole-init",
				"image": inj.config.InitImage,
				"securityContext": map[string]interface{}{
					"runAsUser":    26248,
					"capabilities": map[string]interface{}{"add": []string{"NET_ADMIN"}},
				},
				"env": []map[string]interface{}{
					{"name": "DIPOLE_PROXY_UID", "value": "65534"},
					{"name": "DIPOLE_PROXY_PORT", "value": "15001"},
				},
			},
		})
	}

	// Sidecar container env vars
	sidecarEnv := []map[string]interface{}{
		{"name": "DIPOLE_AGENT_CONTROLLER", "value": inj.config.ControllerURL},
		{"name": "DIPOLE_TENANT_ID", "value": inj.config.TenantID},
		{"name": "K8S_POD_NAME", "valueFrom": map[string]interface{}{"fieldRef": map[string]string{"fieldPath": "metadata.name"}}},
		{"name": "K8S_NAMESPACE", "valueFrom": map[string]interface{}{"fieldRef": map[string]string{"fieldPath": "metadata.namespace"}}},
	}

	// Cloud gRPC endpoint for policy evaluation
	if inj.config.CloudGRPCEndpoint != "" {
		sidecarEnv = append(sidecarEnv, map[string]interface{}{"name": "DIPOLE_CLOUD_GRPC_ENDPOINT", "value": inj.config.CloudGRPCEndpoint})
	}

	if rsID := annotations["dipolehq.com/resource-set-id"]; rsID != "" {
		sidecarEnv = append(sidecarEnv, map[string]interface{}{"name": "DIPOLE_RESOURCE_SET_ID", "value": rsID})
	}
	if clientID := annotations["dipolehq.com/client-id"]; clientID != "" {
		sidecarEnv = append(sidecarEnv, map[string]interface{}{"name": "CLIENT_WORKLOAD_ID", "value": clientID})
	}

	for k, v := range annotations {
		if strings.HasPrefix(k, "dipolehq.com/agent-proxy-env-") {
			varName := strings.TrimPrefix(k, "dipolehq.com/agent-proxy-env-")
			sidecarEnv = append(sidecarEnv, map[string]interface{}{"name": varName, "value": v})
		}
	}

	// Sidecar container
	patches = append(patches, jsonPatch{
		Op:   "add",
		Path: "/spec/containers/-",
		Value: map[string]interface{}{
			"name":            "dipole-agent-proxy",
			"image":           inj.config.ProxyImage,
			"securityContext": map[string]interface{}{"runAsUser": 65534},
			"ports": []map[string]interface{}{
				{"containerPort": 9099, "name": "metrics", "protocol": "TCP"},
			},
			"env": sidecarEnv,
		},
	})

	// Explicit mode: add HTTP_PROXY/HTTPS_PROXY to ALL existing containers
	if steeringMode == "explicit" {
		for i := 0; i < containerCount; i++ {
			patches = append(patches, jsonPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/env/-", i),
				Value: map[string]interface{}{"name": "HTTP_PROXY", "value": "http://localhost:8000"},
			})
			patches = append(patches, jsonPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/env/-", i),
				Value: map[string]interface{}{"name": "HTTPS_PROXY", "value": "http://localhost:8000"},
			})
		}
	}

	return patches
}

func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "dipole-agent-injector"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"dipole-agent-injector", "dipole-agent-injector.dipole-system", "dipole-agent-injector.dipole-system.svc"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
