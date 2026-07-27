/*
kerberos_validator.go verifies Kerberos-attested workload identity via
Controller-signed HMAC-SHA256 attestation documents.

The Agent Controller validates the Kerberos ticket locally (it holds the keytab)
and signs an attestation {principal, realm, domain, timestamp} with HMAC-SHA256
using a shared secret established during registration. Cloud verifies the
signature and checks timestamp freshness (< 5 minutes) to prevent replay.
*/
package trust

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const kerberosTimestampMaxAge = 5 * time.Minute

type KerberosValidator struct {
	controllerRepo repository.AgentControllerRepository
}

func NewKerberosValidator(controllerRepo repository.AgentControllerRepository) *KerberosValidator {
	return &KerberosValidator{controllerRepo: controllerRepo}
}

type kerberosConfig struct {
	MatchRules []MatchRule `json:"match_rules"`
}

type kerberosPayload struct {
	Principal string `json:"principal"`
	Realm     string `json:"realm"`
	Domain    string `json:"domain"`
	Timestamp int64  `json:"timestamp"`
}

func (v *KerberosValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.KerberosAttestation == nil {
		// Fallback: extract from environment map for backward compat during migration
		if evidence.Environment == nil {
			return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
		}
		attestPayload := evidence.Environment["KERBEROS_ATTESTATION_PAYLOAD"]
		attestSig := evidence.Environment["KERBEROS_ATTESTATION_SIGNATURE"]
		controllerID := evidence.Environment["KERBEROS_CONTROLLER_ID"]
		if attestPayload == "" || attestSig == "" || controllerID == "" {
			return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
		}
		evidence.KerberosAttestation = &domain.KerberosAttestation{
			Payload:      attestPayload,
			Signature:    attestSig,
			ControllerID: controllerID,
		}
	}

	attest := evidence.KerberosAttestation

	// Parse and validate controller ID
	controllerUUID, err := uuid.Parse(attest.ControllerID)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Look up controller to get shared secret
	controller, err := v.controllerRepo.GetByID(ctx, provider.TenantID, controllerUUID)
	if err != nil || controller == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if controller.SharedSecret == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	secretBytes, err := hex.DecodeString(controller.SharedSecret)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Verify HMAC-SHA256 signature
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(attest.Payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(attest.Signature), []byte(expectedSig)) {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Decode and validate the verified payload
	payloadBytes, err := base64.StdEncoding.DecodeString(attest.Payload)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	var payload kerberosPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Anti-replay: reject if timestamp is too old
	attestTime := time.Unix(payload.Timestamp, 0)
	if time.Since(attestTime) > kerberosTimestampMaxAge {
		return &domain.TrustProviderResult{Attested: false, Reason: "AttestationExpired"}, nil
	}

	// Parse config match rules
	var cfg kerberosConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse kerberos config: %w", err)
	}

	claims := map[string]interface{}{
		"principal": payload.Principal,
		"realm":     payload.Realm,
		"domain":    payload.Domain,
		"sourceIp":  evidence.SourceIP,
	}

	passed, failure := EvaluateMatchRules(cfg.MatchRules, claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true}, nil
}
