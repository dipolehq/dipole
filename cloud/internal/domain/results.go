// Package domain defines core business entities and value objects.
package domain

import "github.com/google/uuid"

type EvaluationResult struct {
	Allowed          bool
	DenyReason       string
	Credential       *Credential
	InjectionConfig  *InjectionConfig
	PolicyID         uuid.UUID
	ClientWorkloadID uuid.UUID
	ServerWorkloadID uuid.UUID
	TrustResults     []TrustProviderResult
}

type Credential struct {
	Type               string
	Value              string
	Username           string
	Password           string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSSessionToken    string
	ExpiresAt          int64
}

type InjectionConfig struct {
	Method     string
	HeaderName string
	Prefix     string
	QueryParam string
	Scheme     string
}

type TrustProviderResult struct {
	ProviderID    uuid.UUID
	ProviderName  string
	Attested      bool
	Reason        string
	Attribute     string
	ExpectedValue string
	ActualValue   string
	Claims        map[string]interface{}
}
