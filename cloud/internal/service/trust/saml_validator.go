/*
saml_validator.go verifies SAMLv2 Response assertions from identity providers.

SAMLValidator parses the IdP signing certificate from configured metadata
(URL or XML), validates the SAML response XML signature, and evaluates match
rules against assertion claims (aud, iss, sub).

Used primarily for MCP Authorization Server flows where human users
authenticate via SAML. Not triggered from the standard proxy path.
*/
package trust

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type SAMLValidator struct {
	httpClient *http.Client
}

func NewSAMLValidator() *SAMLValidator {
	return &SAMLValidator{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: newSSRFSafeTransport(),
		},
	}
}

type samlConfig struct {
	IdPMetadataURL string `json:"samlMetadataUrl"`
	IdPMetadataXML string `json:"samlMetadataXml"`
}

func (v *SAMLValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Environment == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	samlResponse := evidence.Environment["SAML_RESPONSE"]
	if samlResponse == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg samlConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse saml config: %w", err)
	}

	idpCert, err := v.getIdPCertificate(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("get IdP certificate: %w", err)
	}

	claims, expired, err := v.validateSAMLResponse(samlResponse, idpCert)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}
	if expired {
		return &domain.TrustProviderResult{Attested: false, Reason: "AssertionExpired"}, nil
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

func (v *SAMLValidator) getIdPCertificate(ctx context.Context, cfg samlConfig) (*x509.Certificate, error) {
	var metadataBytes []byte

	if cfg.IdPMetadataXML != "" {
		metadataBytes = []byte(cfg.IdPMetadataXML)
	} else if cfg.IdPMetadataURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.IdPMetadataURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := v.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("metadata fetch returned %d", resp.StatusCode)
		}
		metadataBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no IdP metadata configured")
	}

	var descriptor saml.EntityDescriptor
	if err := xml.Unmarshal(metadataBytes, &descriptor); err != nil {
		return nil, fmt.Errorf("parse IdP metadata XML: %w", err)
	}

	for _, kd := range descriptor.IDPSSODescriptors {
		for _, keyDesc := range kd.KeyDescriptors {
			if keyDesc.Use == "signing" || keyDesc.Use == "" {
				for _, cert := range keyDesc.KeyInfo.X509Data.X509Certificates {
					certBytes, err := base64.StdEncoding.DecodeString(cert.Data)
					if err != nil {
						continue
					}
					parsed, err := x509.ParseCertificate(certBytes)
					if err != nil {
						continue
					}
					return parsed, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no signing certificate found in IdP metadata")
}

// samlClockSkew tolerates minor clock drift between the IdP and this server
// when checking assertion validity windows, matching the default used by the
// crewjam/saml Service Provider implementation.
const samlClockSkew = 180 * time.Second

func (v *SAMLValidator) validateSAMLResponse(responseBase64 string, idpCert *x509.Certificate) (claims map[string]interface{}, expired bool, err error) {
	responseBytes, err := base64.StdEncoding.DecodeString(responseBase64)
	if err != nil {
		return nil, false, fmt.Errorf("decode SAML response: %w", err)
	}

	// Verify XML digital signature using IdP certificate
	if err := verifySAMLSignature(responseBytes, idpCert); err != nil {
		return nil, false, fmt.Errorf("signature verification failed: %w", err)
	}

	var response saml.Response
	if err := xml.Unmarshal(responseBytes, &response); err != nil {
		return nil, false, fmt.Errorf("parse SAML response XML: %w", err)
	}

	// Extract claims from the SAML assertion
	claims = map[string]interface{}{}

	if response.Issuer != nil {
		claims["iss"] = response.Issuer.Value
	}

	if response.Destination != "" {
		claims["aud"] = response.Destination
	}

	if response.Assertion != nil {
		if response.Assertion.Subject != nil && response.Assertion.Subject.NameID != nil {
			claims["sub"] = response.Assertion.Subject.NameID.Value
		}
		if response.Assertion.Conditions != nil {
			for _, ar := range response.Assertion.Conditions.AudienceRestrictions {
				claims["aud"] = ar.Audience.Value
			}
			if assertionExpired(response.Assertion.Conditions.NotBefore, response.Assertion.Conditions.NotOnOrAfter) {
				return claims, true, nil
			}
		}

		for _, stmt := range response.Assertion.AttributeStatements {
			for _, attr := range stmt.Attributes {
				if len(attr.Values) > 0 {
					claims[attr.Name] = attr.Values[0].Value
				}
			}
		}
	}

	return claims, false, nil
}

// assertionExpired reports whether now falls outside the assertion's
// [NotBefore, NotOnOrAfter) validity window, allowing samlClockSkew of
// tolerance in each direction. A captured SAMLResponse can otherwise be
// replayed for as long as the IdP's validity window remains open, since
// nothing else in this flow is bound to a single use of the assertion.
// Zero-value times (field absent from the assertion) are not enforced.
func assertionExpired(notBefore, notOnOrAfter time.Time) bool {
	now := time.Now().UTC()
	if !notBefore.IsZero() && notBefore.Add(-samlClockSkew).After(now) {
		return true
	}
	if !notOnOrAfter.IsZero() && notOnOrAfter.Add(samlClockSkew).Before(now) {
		return true
	}
	return false
}

func verifySAMLSignature(responseXML []byte, idpCert *x509.Certificate) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(responseXML); err != nil {
		return fmt.Errorf("parse XML: %w", err)
	}

	sigEl := doc.Root().FindElement("//Signature")
	if sigEl == nil {
		// Also check within Assertion
		sigEl = doc.Root().FindElement("./Assertion/Signature")
	}
	if sigEl == nil {
		return fmt.Errorf("no Signature element found")
	}

	certStore := dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{idpCert}}
	validationCtx := dsig.NewDefaultValidationContext(&certStore)

	_, err := validationCtx.Validate(doc.Root())
	return err
}
