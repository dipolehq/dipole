package main

import "testing"

func TestBuildClientBlock_Certificate_WithAttestation(t *testing.T) {
	attest := &certificateAttestation{
		CertificatePEM:  "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		Timestamp:       1700000000,
		SignatureBase64: "AQIDBA==",
	}

	block := buildClientBlock("", "certificate", attest)

	cert, ok := block["certificate"].(map[string]any)
	if !ok {
		t.Fatal("expected block[\"certificate\"] to be present")
	}
	if cert["certificatePem"] != attest.CertificatePEM {
		t.Errorf("certificatePem mismatch: got %v", cert["certificatePem"])
	}
	if cert["timestamp"] != attest.Timestamp {
		t.Errorf("timestamp mismatch: got %v", cert["timestamp"])
	}
	if cert["signature"] != attest.SignatureBase64 {
		t.Errorf("signature mismatch: got %v", cert["signature"])
	}
}

func TestBuildClientBlock_Certificate_NilAttestation_EmptyBlock(t *testing.T) {
	block := buildClientBlock("", "certificate", nil)

	if len(block) != 0 {
		t.Errorf("expected empty block when certificate attestation is nil, got %v", block)
	}
}

func TestBuildClientBlock_UnknownIdentityType_FallsBackToOIDC(t *testing.T) {
	block := buildClientBlock("my-token", "oidc_id_token", nil)

	oidc, ok := block["oidc"].(map[string]any)
	if !ok {
		t.Fatal("expected block[\"oidc\"] to be present for default case")
	}
	if oidc["identityToken"] != "my-token" {
		t.Errorf("identityToken mismatch: got %v", oidc["identityToken"])
	}
}
