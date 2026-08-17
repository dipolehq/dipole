/*
Package cloudclient tests the TLS transport credential building for the Cloud gRPC connection.
*/
package cloudclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestBuildTransportCredentials_Insecure(t *testing.T) {
	creds, err := buildTransportCredentials("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds == nil {
		t.Fatal("expected non-nil credentials")
	}
	info := creds.Info()
	if info.SecurityProtocol != "insecure" {
		t.Errorf("expected 'insecure' security protocol, got %q", info.SecurityProtocol)
	}
}

func TestBuildTransportCredentials_SystemRoots(t *testing.T) {
	creds, err := buildTransportCredentials("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds == nil {
		t.Fatal("expected non-nil credentials")
	}
	info := creds.Info()
	if info.SecurityProtocol != "tls" {
		t.Errorf("expected 'tls' security protocol, got %q", info.SecurityProtocol)
	}
}

func TestBuildTransportCredentials_CustomCA(t *testing.T) {
	caPEM := generateTestCAPEM(t)
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, caPEM, 0600); err != nil {
		t.Fatal(err)
	}

	creds, err := buildTransportCredentials(caPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info := creds.Info()
	if info.SecurityProtocol != "tls" {
		t.Errorf("expected 'tls', got %q", info.SecurityProtocol)
	}
}

func TestBuildTransportCredentials_InvalidPath(t *testing.T) {
	_, err := buildTransportCredentials("/nonexistent/ca.pem", false)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestBuildTransportCredentials_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.pem")
	os.WriteFile(badPath, []byte("not a valid PEM"), 0600)

	_, err := buildTransportCredentials(badPath, false)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func generateTestCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test CA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1 * time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func TestCloudClient_AttachSessionToken_NoTokenYet_LeavesContextUnchanged(t *testing.T) {
	c := &CloudClient{}
	c.sessionToken.Store("")

	ctx := c.attachSessionToken(context.Background())
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok && len(md.Get("authorization")) > 0 {
		t.Errorf("expected no authorization metadata before a session token is issued, got %v", md.Get("authorization"))
	}
}

func TestCloudClient_AttachSessionToken_AttachesBearerToken(t *testing.T) {
	c := &CloudClient{}
	c.sessionToken.Store("my-session-token")

	ctx := c.attachSessionToken(context.Background())
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata to be set")
	}
	values := md.Get("authorization")
	if len(values) != 1 || values[0] != "Bearer my-session-token" {
		t.Errorf("authorization metadata = %v, want [\"Bearer my-session-token\"]", values)
	}
}

func TestCloudClient_AttachSessionToken_ReflectsMostRecentlyStoredToken(t *testing.T) {
	c := &CloudClient{}
	c.sessionToken.Store("first-token")
	c.sessionToken.Store("second-token")

	ctx := c.attachSessionToken(context.Background())
	md, _ := metadata.FromOutgoingContext(ctx)
	values := md.Get("authorization")
	if len(values) != 1 || values[0] != "Bearer second-token" {
		t.Errorf("authorization metadata = %v, want [\"Bearer second-token\"] (most recent token)", values)
	}
}
