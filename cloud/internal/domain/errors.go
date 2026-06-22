package domain

import "errors"

var (
	ErrPolicyNotFound            = errors.New("no matching policy found")
	ErrAttestationFailed         = errors.New("attestation failed")
	ErrConditionFailed           = errors.New("access condition failed")
	ErrCredentialRetrievalFailed = errors.New("credential retrieval failed")
	ErrTenantNotFound            = errors.New("tenant not found")
	ErrInactivePolicy            = errors.New("policy is inactive")
	ErrInvalidIdentifier         = errors.New("invalid identifier")
	ErrEncryptionFailed          = errors.New("encryption failed")
)
