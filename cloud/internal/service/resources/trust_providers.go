/*
Package resources — trust_providers.go implements business logic and
validation for Trust Providers.
*/
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/trust"
)

// TrustProviderService is stateless with respect to tenant. The Edge SDK Client
// ID and Audience it generates embed the OWNING tenant's subdomain — these are
// the values a customer copies into their GitHub Actions workflow or CLI config
// — so the subdomain is looked up per call from the tenant that's creating the
// provider, not captured once at startup.
type TrustProviderService struct {
	repo          repository.TrustProviderRepository
	tenants       repository.TenantRepository
	region        string
	hosts         *platform.Hosts
	encryptionKey []byte
	limiter       *service.PlanLimiter
}

func NewTrustProviderService(
	repo repository.TrustProviderRepository,
	tenants repository.TenantRepository,
	region string,
	hosts *platform.Hosts,
	encryptionKey []byte,
) *TrustProviderService {
	return &TrustProviderService{repo: repo, tenants: tenants, region: region, hosts: hosts, encryptionKey: encryptionKey}
}

func (s *TrustProviderService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *TrustProviderService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.TrustProvider, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *TrustProviderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if tp == nil {
		return nil, domain.ErrNotFound
	}
	return tp, nil
}

func (s *TrustProviderService) Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceTrustProviders); err != nil {
			return err
		}
	}
	normalizeTrustProvider(tp)
	if err := validateTrustProvider(tp); err != nil {
		return err
	}

	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("resolve tenant for edge sdk identifiers: %w", err)
	}
	if tenant == nil {
		return domain.ErrNotFound
	}

	// These two values are immutable after creation and get copied by the
	// customer into their CI/CD config, so refuse to persist a malformed one.
	audience := s.hosts.IdentityIssuerURL(tenant.Subdomain)
	if audience == "" {
		return fmt.Errorf(
			"cannot generate Edge SDK audience: tenant subdomain %q is not a valid hostname label",
			tenant.Subdomain)
	}

	tp.ID = uuid.New()
	tp.EdgeSDKClientID = trust.GenerateEdgeSDKClientID(s.region, tenant.Subdomain, tp.Type, tp.ID)
	tp.EdgeSDKAudience = audience

	if err := s.extractAndEncryptSymmetricKey(tp); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, tp)
}

func (s *TrustProviderService) Update(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	normalizeTrustProvider(tp)
	if err := validateTrustProvider(tp); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, tp.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	tp.EdgeSDKClientID = existing.EdgeSDKClientID
	tp.EdgeSDKAudience = existing.EdgeSDKAudience

	if err := s.extractAndEncryptSymmetricKey(tp); err != nil {
		return err
	}
	// The symmetric key is never returned to API clients (see
	// extractAndEncryptSymmetricKey), so an update request that doesn't
	// include a new one is not "the customer wants to clear it" — it's
	// "the client never had it to resend." Preserve the previously
	// stored encrypted value unless this request actually supplied a new
	// plaintext key.
	if tp.SymmetricKeyEncrypted == nil {
		tp.SymmetricKeyEncrypted = existing.SymmetricKeyEncrypted
	}
	return s.repo.Update(ctx, tenantID, tp)
}

// extractAndEncryptSymmetricKey removes the "symmetricKey" field from a
// symmetric_key-attestation OIDC trust provider's Config (a real HMAC
// shared secret, historically stored there in plaintext) and encrypts it
// into SymmetricKeyEncrypted instead. Config keeps every other field
// (attestationMethod, oidcEndpoint, jwks, publicKey) exactly as supplied —
// none of those are secret. If Config has no "symmetricKey" key at all
// (any other attestation method, or an update that intentionally omits
// it), this is a no-op: tp.SymmetricKeyEncrypted stays nil, and the caller
// decides what nil means for that operation (Create: no secret; Update:
// caller re-applies the previously stored value).
func (s *TrustProviderService) extractAndEncryptSymmetricKey(tp *domain.TrustProvider) error {
	if len(tp.Config) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(tp.Config, &raw); err != nil {
		return domain.NewValidationError("config", "config must be a JSON object")
	}

	rawKey, ok := raw["symmetricKey"]
	if !ok {
		return nil
	}

	var symmetricKey string
	if err := json.Unmarshal(rawKey, &symmetricKey); err != nil {
		return domain.NewValidationError("config.symmetricKey", "symmetricKey must be a string")
	}

	delete(raw, "symmetricKey")
	strippedConfig, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("re-marshal trust provider config after stripping symmetricKey: %w", err)
	}
	tp.Config = strippedConfig

	if symmetricKey == "" {
		return nil
	}

	encrypted, err := crypto.Encrypt([]byte(symmetricKey), s.encryptionKey)
	if err != nil {
		return domain.NewValidationError("config.symmetricKey", "failed to encrypt symmetric key")
	}
	tp.SymmetricKeyEncrypted = encrypted
	return nil
}

func (s *TrustProviderService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeTrustProvider(tp *domain.TrustProvider) {
	tp.Name = strings.TrimSpace(tp.Name)
	tp.Description = strings.TrimSpace(tp.Description)
}

func validateTrustProvider(tp *domain.TrustProvider) error {
	if tp.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if tp.Type == "" {
		return domain.NewValidationError("type", "trust provider type is required")
	}
	return nil
}

// Edge SDK identifier generation lives in service/trust/edge_sdk.go. It used to
// be duplicated here (with a second, unused exported copy in that package),
// which is how the two drifted apart during the subdomain pattern change.
