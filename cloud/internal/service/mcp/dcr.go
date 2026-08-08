/*
Package mcp implements the MCP Authorization Server business logic.

DCRService handles Dynamic Client Registration (RFC 7591). MCP clients
self-register at runtime by POSTing redirect_uris and grant_types, receiving
a unique client_id for subsequent authorization requests.
*/
package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMissingRedirectURIs = errors.New("redirect_uris is required and must be non-empty")
	ErrMissingGrantType    = errors.New("grant_types must include authorization_code")
	ErrMissingResponseType = errors.New("response_types must include code")
)

type RegisteredClient struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	RedirectURIs []string
	GrantTypes   []string
	CreatedAt    time.Time
}

type MCPClientRepository interface {
	Create(ctx context.Context, client *RegisteredClient) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*RegisteredClient, error)
}

type DCRRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ClientName              string   `json:"client_name"`
}

type DCRResponse struct {
	ClientID                string   `json:"client_id"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type DCRService struct {
	repo MCPClientRepository
}

func NewDCRService(repo MCPClientRepository) *DCRService {
	return &DCRService{repo: repo}
}

func (s *DCRService) Register(ctx context.Context, tenantID uuid.UUID, req *DCRRequest) (*DCRResponse, error) {
	// RFC 7591 §2: when grant_types is omitted, the default is
	// ["authorization_code"]; when response_types is omitted, the default is
	// ["code"]. Claude.ai omits both, so we must apply defaults before
	// validating rather than rejecting.
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}

	if err := validateDCRRequest(req); err != nil {
		return nil, err
	}

	client := &RegisteredClient{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RedirectURIs: req.RedirectURIs,
		GrantTypes:   req.GrantTypes,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "none"
	}

	return &DCRResponse{
		ClientID:                client.ID.String(),
		RedirectURIs:            client.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		TokenEndpointAuthMethod: authMethod,
	}, nil
}

func (s *DCRService) GetClient(ctx context.Context, tenantID uuid.UUID, clientID uuid.UUID) (*RegisteredClient, error) {
	return s.repo.GetByID(ctx, tenantID, clientID)
}

func validateDCRRequest(req *DCRRequest) error {
	if len(req.RedirectURIs) == 0 {
		return ErrMissingRedirectURIs
	}
	if !contains(req.GrantTypes, "authorization_code") {
		return ErrMissingGrantType
	}
	if !contains(req.ResponseTypes, "code") {
		return ErrMissingResponseType
	}
	return nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
