// Package domain defines core business entities and value objects.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ClientWorkload struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Name          string
	Description   string
	IsActive      bool
	Identifiers   []Identifier
	Tags          []Tag
	ResourceSetID *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PolicyCount   int
}

type ServerWorkload struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	Name              string
	Description       string
	IsActive          bool
	Host              string
	Port              int
	AppProtocol       string
	TransportProtocol string
	TLS               bool
	TLSVerification   string
	RequestedPort     int
	RequestedTLS      bool
	AuthMethod        string
	AuthScheme        string
	AuthConfig        map[string]string
	HTTPHeaders       map[string]string
	URLPath           string
	Tags              []Tag
	ResourceSetID     *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	PolicyCount       int
}
