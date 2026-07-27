package main

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestPatchWebhookCABundle_NotInCluster_Skips(t *testing.T) {
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("KUBERNETES_SERVICE_PORT")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := patchWebhookCABundle("test-webhook", []byte{1, 2, 3}, logger)
	if err != nil {
		t.Fatalf("expected no error when not in cluster, got: %v", err)
	}
}

func TestPatchJSON_Format(t *testing.T) {
	certDER := []byte("test-certificate-bytes")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	expectedBundle := base64.StdEncoding.EncodeToString(caPEM)
	patch := fmt.Sprintf(`[{"op":"replace","path":"/webhooks/0/clientConfig/caBundle","value":"%s"}]`, expectedBundle)

	if patch[0] != '[' {
		t.Error("patch should be a JSON array")
	}
	if !strings.Contains(patch, `"op":"replace"`) {
		t.Error("patch should have op=replace")
	}
	if !strings.Contains(patch, `"/webhooks/0/clientConfig/caBundle"`) {
		t.Error("patch should target caBundle path")
	}
	if !strings.Contains(patch, expectedBundle) {
		t.Error("patch should contain the base64-encoded PEM cert")
	}

	// Verify the bundle decodes to valid PEM
	decoded, err := base64.StdEncoding.DecodeString(expectedBundle)
	if err != nil {
		t.Fatalf("bundle should be valid base64: %v", err)
	}
	block, _ := pem.Decode(decoded)
	if block == nil {
		t.Fatal("decoded bundle should be valid PEM")
	}
	if block.Type != "CERTIFICATE" {
		t.Errorf("PEM block type should be CERTIFICATE, got %s", block.Type)
	}
}

func TestGenerateSelfSignedCert_ReturnsCertAndDER(t *testing.T) {
	cert, der, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Error("expected at least one certificate in chain")
	}
	if len(der) == 0 {
		t.Error("expected non-empty DER bytes")
	}
	if len(cert.Certificate[0]) != len(der) {
		t.Error("DER bytes should match first cert in chain")
	}
}
