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
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type samlConfig struct {
	IdPMetadataURL string      `json:"idp_metadata_url"`
	IdPMetadataXML string      `json:"idp_metadata_xml"`
	MatchRules     []MatchRule `json:"match_rules"`
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

	claims, err := v.validateSAMLResponse(samlResponse, idpCert)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
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

func (v *SAMLValidator) validateSAMLResponse(responseBase64 string, idpCert *x509.Certificate) (map[string]interface{}, error) {
	responseBytes, err := base64.StdEncoding.DecodeString(responseBase64)
	if err != nil {
		return nil, fmt.Errorf("decode SAML response: %w", err)
	}

	// Verify XML digital signature using IdP certificate
	if err := verifySAMLSignature(responseBytes, idpCert); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	var response saml.Response
	if err := xml.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("parse SAML response XML: %w", err)
	}

	// Extract claims from the SAML assertion
	claims := map[string]interface{}{}

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
		}
		for _, stmt := range response.Assertion.AttributeStatements {
			for _, attr := range stmt.Attributes {
				if len(attr.Values) > 0 {
					claims[attr.Name] = attr.Values[0].Value
				}
			}
		}
	}

	return claims, nil
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
