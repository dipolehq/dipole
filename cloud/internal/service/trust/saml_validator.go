// saml_validator.go verifies SAMLv2 Response assertions, binding XML-DSig signature validation to the specific Response/Assertion element.
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
	"github.com/russellhaering/goxmldsig/etreeutils"

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

const samlClockSkew = 180 * time.Second

func (v *SAMLValidator) validateSAMLResponse(responseBase64 string, idpCert *x509.Certificate) (claims map[string]interface{}, expired bool, err error) {
	responseBytes, err := base64.StdEncoding.DecodeString(responseBase64)
	if err != nil {
		return nil, false, fmt.Errorf("decode SAML response: %w", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(responseBytes); err != nil {
		return nil, false, fmt.Errorf("parse XML: %w", err)
	}
	if doc.Root() == nil {
		return nil, false, fmt.Errorf("invalid SAML response: no root element")
	}

	assertionEls := doc.Root().FindElements("./Assertion")
	if len(assertionEls) == 0 {
		return nil, false, fmt.Errorf("no Assertion element found in SAML response")
	}
	if len(assertionEls) > 1 {
		return nil, false, fmt.Errorf("SAML response contains %d Assertion elements, expected exactly 1", len(assertionEls))
	}
	assertionEl := assertionEls[0]

	if err := verifyAndBindSAMLSignature(doc.Root(), assertionEl, idpCert); err != nil {
		return nil, false, fmt.Errorf("signature verification failed: %w", err)
	}

	var assertion saml.Assertion
	if err := unmarshalElement(assertionEl, &assertion); err != nil {
		return nil, false, fmt.Errorf("parse Assertion XML: %w", err)
	}

	var response saml.Response
	if err := unmarshalElement(doc.Root(), &response); err != nil {
		return nil, false, fmt.Errorf("parse SAML response XML: %w", err)
	}

	claims = map[string]interface{}{}

	if response.Issuer != nil {
		claims["iss"] = response.Issuer.Value
	}

	if response.Destination != "" {
		claims["aud"] = response.Destination
	}

	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		claims["sub"] = assertion.Subject.NameID.Value
	}
	if assertion.Conditions != nil {
		for _, ar := range assertion.Conditions.AudienceRestrictions {
			claims["aud"] = ar.Audience.Value
		}
		if assertionExpired(assertion.Conditions.NotBefore, assertion.Conditions.NotOnOrAfter) {
			return claims, true, nil
		}
	}

	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) > 0 {
				claims[attr.Name] = attr.Values[0].Value
			}
		}
	}

	return claims, false, nil
}

func unmarshalElement(el *etree.Element, v interface{}) error {
	namespaces := map[string]string{}
	for _, descendant := range el.FindElements("//*") {
		if ns := descendant.NamespaceURI(); ns != "" {
			namespaces[descendant.Space] = ns
		}
	}

	doc := etree.NewDocument()
	doc.SetRoot(el.Copy())
	for prefix, uri := range namespaces {
		doc.Root().CreateAttr("xmlns:"+prefix, uri)
	}

	buf, err := doc.WriteToBytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(buf, v)
}

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

func verifyAndBindSAMLSignature(responseEl, assertionEl *etree.Element, idpCert *x509.Certificate) error {
	certStore := dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{idpCert}}

	if assertionEl.FindElement("./Signature") != nil {
		return validateScopedSignature(assertionEl, &certStore)
	}
	if responseEl.FindElement("./Signature") != nil {
		return validateScopedSignature(responseEl, &certStore)
	}
	return fmt.Errorf("no Signature element found on the Response or its Assertion")
}

func validateScopedSignature(el *etree.Element, certStore dsig.X509CertificateStore) error {
	parentCtx, err := etreeutils.NSBuildParentContext(el)
	if err != nil {
		return fmt.Errorf("build namespace context: %w", err)
	}
	ctx, err := parentCtx.SubContext(el)
	if err != nil {
		return fmt.Errorf("build namespace context: %w", err)
	}
	detached, err := etreeutils.NSDetatch(ctx, el)
	if err != nil {
		return fmt.Errorf("detach element for signature scoping: %w", err)
	}

	validationCtx := dsig.NewDefaultValidationContext(certStore)
	_, err = validationCtx.Validate(detached)
	return err
}
