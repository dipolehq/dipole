/*
auth_code.go manages OAuth authorization codes with PKCE binding.

AuthCodeStore creates short-lived (5 min), single-use authorization codes
tied to a code_challenge. Codes are stored in-memory for the MVP; the DB
migration exists for future persistence.
*/
package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCodeNotFound = errors.New("authorization code not found")
	ErrCodeExpired  = errors.New("authorization code expired")
	ErrCodeUsed     = errors.New("authorization code already used")
)

type PendingAuthCode struct {
	Code          string
	ClientID      uuid.UUID
	TenantID      uuid.UUID
	RedirectURI   string
	CodeChallenge string
	Scope         string
	UserClaims    map[string]any
	PolicyID      uuid.UUID
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Used          bool
}

type AuthCodeStore struct {
	mu    sync.RWMutex
	codes map[string]*PendingAuthCode
}

func NewAuthCodeStore() *AuthCodeStore {
	return &AuthCodeStore{codes: make(map[string]*PendingAuthCode)}
}

func (s *AuthCodeStore) Create(clientID, tenantID uuid.UUID, redirectURI, codeChallenge, scope string, userClaims map[string]any, policyID uuid.UUID) string {
	code := generateCode()
	now := time.Now().UTC()

	s.mu.Lock()
	s.codes[code] = &PendingAuthCode{
		Code:          code,
		ClientID:      clientID,
		TenantID:      tenantID,
		RedirectURI:   redirectURI,
		CodeChallenge: codeChallenge,
		Scope:         scope,
		UserClaims:    userClaims,
		PolicyID:      policyID,
		CreatedAt:     now,
		ExpiresAt:     now.Add(5 * time.Minute),
		Used:          false,
	}
	s.mu.Unlock()

	return code
}

func (s *AuthCodeStore) Consume(code string) (*PendingAuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending, ok := s.codes[code]
	if !ok {
		return nil, ErrCodeNotFound
	}
	if pending.Used {
		return nil, ErrCodeUsed
	}
	if time.Now().UTC().After(pending.ExpiresAt) {
		return nil, ErrCodeExpired
	}

	pending.Used = true
	return pending, nil
}

func generateCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
