package domain

import "github.com/google/uuid"

type ClientEvidence struct {
	SourceIP               string
	Hostname               string
	K8s                    *K8sEvidence
	AWS                    *AWSEvidence
	Azure                  *AzureEvidence
	GCP                    *GCPEvidence
	GitHub                 *IdentityTokenEvidence
	GitLab                 *IdentityTokenEvidence
	Terraform              *IdentityTokenEvidence
	Process                *ProcessEvidence
	Environment            map[string]string
	DipoleClientID         string
	RedirectURI            string
	KerberosAttestation    *KerberosAttestation
	CertificateAttestation *CertificateAttestation
}

// CertificateAttestation holds a workload's X.509 certificate plus proof
// that the presenter possesses the certificate's private key: a signature,
// produced by that private key, over a canonical challenge binding the
// certificate and a timestamp. A certificate alone is public information
// (visible in TLS handshakes, logs, etc.) and proves nothing on its own —
// this signature is what distinguishes "I hold this identity" from "I saw
// this identity somewhere."
//
// See buildCertificateChallenge in certificate_validator.go for the exact
// canonical byte format the signature covers.
type CertificateAttestation struct {
	CertificatePEM string
	Timestamp      int64 // unix seconds, UTC, as claimed by the signer
	Signature      []byte
}

// KerberosAttestation holds an HMAC-SHA256 signed attestation from the Agent
// Controller after it has validated the Kerberos ticket locally. Cloud verifies
// the signature using the shared secret established during controller registration.
type KerberosAttestation struct {
	Payload      string // base64-encoded JSON: {principal, realm, domain, timestamp}
	Signature    string // HMAC-SHA256 hex signature over payload
	ControllerID string // UUID of the signing Agent Controller
}

type K8sEvidence struct {
	ServiceAccountToken string
	Namespace           string
	PodName             string
	ServiceAccountName  string
	ServiceAccountUID   string
}

type AWSEvidence struct {
	InstanceIdentityDocument  string
	InstanceIdentitySignature string
	STSGetCallerIdentity      *STSGetCallerIdentity
}

type STSGetCallerIdentity struct {
	SignedHeaders map[string]string
	Region        string
}

type AzureEvidence struct {
	Encoding  string
	Signature string
	Nonce     string
}

type GCPEvidence struct {
	IdentityToken string
}

type IdentityTokenEvidence struct {
	IdentityToken string
}

type ProcessEvidence struct {
	Name        string
	Path        string
	CommandLine string
	UserName    string
}

type ServerTarget struct {
	Host          string
	Port          int
	ResourceSetID *uuid.UUID
}
