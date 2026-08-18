// kerberos_validator.go verifies Kerberos-attested workload identity via Controller-signed HMAC-SHA256 attestation documents, checking timestamp freshness to prevent replay.
package trust

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const kerberosTimestampMaxAge = 5 * time.Minute

type KerberosValidator struct {
	controllerRepo repository.AgentControllerRepository
	encryptionKey  []byte
}

func NewKerberosValidator(controllerRepo repository.AgentControllerRepository, encryptionKey []byte) *KerberosValidator {
	return &KerberosValidator{controllerRepo: controllerRepo, encryptionKey: encryptionKey}
}

type kerberosPayload struct {
	Principal string `json:"principal"`
	Realm     string `json:"realm"`
	Domain    string `json:"domain"`
	Timestamp int64  `json:"timestamp"`
}

func (v *KerberosValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.KerberosAttestation == nil {
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

	controllerUUID, err := uuid.Parse(attest.ControllerID)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	controller, err := v.controllerRepo.GetByID(ctx, provider.TenantID, controllerUUID)
	if err != nil || controller == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if len(controller.SharedSecretEncrypted) == 0 {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	secretHex, err := crypto.Decrypt(controller.SharedSecretEncrypted, v.encryptionKey)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	secretBytes, err := hex.DecodeString(string(secretHex))
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(attest.Payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(attest.Signature), []byte(expectedSig)) {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(attest.Payload)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	var payload kerberosPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	attestTime := time.Unix(payload.Timestamp, 0)
	if time.Since(attestTime) > kerberosTimestampMaxAge {
		return &domain.TrustProviderResult{Attested: false, Reason: "AttestationExpired"}, nil
	}

	claims := map[string]interface{}{
		"principal": payload.Principal,
		"realm":     payload.Realm,
		"domain":    payload.Domain,
		"sourceIp":  evidence.SourceIP,
	}

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: claims}, nil
}
