package domain

import (
	"errors"
	"fmt"
)

var (
	ErrPolicyNotFound            = errors.New("no matching policy found")
	ErrAttestationFailed         = errors.New("attestation failed")
	ErrConditionFailed           = errors.New("access condition failed")
	ErrCredentialRetrievalFailed = errors.New("credential retrieval failed")
	ErrTenantNotFound            = errors.New("tenant not found")
	ErrInactivePolicy            = errors.New("policy is inactive")
	ErrInvalidIdentifier         = errors.New("invalid identifier")
	ErrEncryptionFailed          = errors.New("encryption failed")
	ErrNotFound                  = errors.New("resource not found")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// RegistrationError represents a client-caused registration failure
// (invalid token, tenant mismatch, controller not found/healthy).
// Server-side errors (DB failures) should NOT use this type.
type RegistrationError struct {
	Message string
}

func (e *RegistrationError) Error() string {
	return e.Message
}

func NewRegistrationError(message string) *RegistrationError {
	return &RegistrationError{Message: message}
}
