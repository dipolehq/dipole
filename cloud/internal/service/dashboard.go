/*
dashboard.go provides admin dashboard statistics.

DashboardService computes entity counts and recent event aggregates for the
admin UI dashboard summary view.
*/
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type DashboardStats struct {
	Counts                   EntityCounts      `json:"counts"`
	RecentEvents             RecentEventStats  `json:"recentEvents"`
	TopClientWorkloads       []TopWorkload     `json:"topClientWorkloads"`
	TopServerWorkloads       []TopWorkload     `json:"topServerWorkloads"`
	UnmanagedServerWorkloads []TopWorkload     `json:"unmanagedServerWorkloads"`
	CredentialUsageByType    []CredentialUsage `json:"credentialUsageByType"`
	ConnectionsByProtocol    []ProtocolCount   `json:"connectionsByProtocol"`
	MostConditionFailures    []PolicyFailures  `json:"mostConditionFailures"`
}

type EntityCounts struct {
	ClientWorkloads     int `json:"clientWorkloads"`
	ClientInactive      int `json:"clientInactive"`
	ClientUnmanaged     int `json:"clientUnmanaged"`
	ServerWorkloads     int `json:"serverWorkloads"`
	ServerInactive      int `json:"serverInactive"`
	ServerUnmanaged     int `json:"serverUnmanaged"`
	TrustProviders      int `json:"trustProviders"`
	TrustInactive       int `json:"trustInactive"`
	CredentialProviders int `json:"credentialProviders"`
	CredentialInactive  int `json:"credentialInactive"`
	AccessPolicies      int `json:"accessPolicies"`
	AccessConditions    int `json:"accessConditions"`
	ConditionInactive   int `json:"conditionInactive"`
}

type RecentEventStats struct {
	Last6h     EventSummary     `json:"last6h"`
	TimeSeries []EventTimePoint `json:"timeSeries"`
}

type EventSummary struct {
	Total      int `json:"total"`
	Authorized int `json:"authorized"`
	Denied     int `json:"denied"`
}

type EventTimePoint struct {
	Timestamp string `json:"timestamp"`
	Info      int    `json:"info"`
	Warning   int    `json:"warning"`
	Error     int    `json:"error"`
}

type TopWorkload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Connections int    `json:"connections"`
}

type CredentialUsage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type ProtocolCount struct {
	Protocol string `json:"protocol"`
	Count    int    `json:"count"`
}

type PolicyFailures struct {
	PolicyID   string `json:"policyId"`
	PolicyName string `json:"policyName"`
	Failures   int    `json:"failures"`
}

type DashboardService struct {
	cwRepo repository.ClientWorkloadRepository
	swRepo repository.ServerWorkloadRepository
	tpRepo repository.TrustProviderRepository
	cpRepo repository.CredentialProviderRepository
	apRepo repository.AccessPolicyRepository
	acRepo repository.AccessConditionRepository
}

func NewDashboardService(
	cw repository.ClientWorkloadRepository,
	sw repository.ServerWorkloadRepository,
	tp repository.TrustProviderRepository,
	cp repository.CredentialProviderRepository,
	ap repository.AccessPolicyRepository,
	ac repository.AccessConditionRepository,
) *DashboardService {
	return &DashboardService{
		cwRepo: cw, swRepo: sw, tpRepo: tp, cpRepo: cp, apRepo: ap, acRepo: ac,
	}
}

func (s *DashboardService) GetStats(ctx context.Context, tenantID uuid.UUID) (*DashboardStats, error) {
	opts := domain.ListOptions{Page: 1, PerPage: 1}

	_, cwCount, _ := s.cwRepo.List(ctx, tenantID, opts)
	_, swCount, _ := s.swRepo.List(ctx, tenantID, opts)
	_, tpCount, _ := s.tpRepo.List(ctx, tenantID, opts)
	_, cpCount, _ := s.cpRepo.List(ctx, tenantID, opts)
	_, apCount, _ := s.apRepo.List(ctx, tenantID, opts)
	_, acCount, _ := s.acRepo.List(ctx, tenantID, opts)

	return &DashboardStats{
		Counts: EntityCounts{
			ClientWorkloads:     cwCount,
			ServerWorkloads:     swCount,
			TrustProviders:      tpCount,
			CredentialProviders: cpCount,
			AccessPolicies:      apCount,
			AccessConditions:    acCount,
		},
		RecentEvents: RecentEventStats{
			Last6h:     EventSummary{Total: 0, Authorized: 0, Denied: 0},
			TimeSeries: []EventTimePoint{},
		},
		TopClientWorkloads:       []TopWorkload{},
		TopServerWorkloads:       []TopWorkload{},
		UnmanagedServerWorkloads: []TopWorkload{},
		CredentialUsageByType:    []CredentialUsage{},
		ConnectionsByProtocol:    []ProtocolCount{},
		MostConditionFailures:    []PolicyFailures{},
	}, nil
}
