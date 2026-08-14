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
	Counts                   EntityCounts        `json:"counts"`
	RecentEvents             RecentEventStats    `json:"recentEvents"`
	TopClientWorkloads       []TopWorkload       `json:"topClientWorkloads"`
	TopServerWorkloads       []TopWorkload       `json:"topServerWorkloads"`
	UnmanagedServerWorkloads []TopWorkload       `json:"unmanagedServerWorkloads"`
	CredentialUsageByType    []CredentialUsage   `json:"credentialUsageByType"`
	ConnectionsByProtocol    []ProtocolCount     `json:"connectionsByProtocol"`
	MostConditionFailures    []PolicyFailures    `json:"mostConditionFailures"`
	AttestationResults       []AttestationResult `json:"attestationResults"`
}

type EntityCounts struct {
	ClientWorkloads        int `json:"clientWorkloads"`
	ClientInactive         int `json:"clientInactive"`
	ClientUnmanaged        int `json:"clientUnmanaged"`
	ServerWorkloads        int `json:"serverWorkloads"`
	ServerInactive         int `json:"serverInactive"`
	ServerUnmanaged        int `json:"serverUnmanaged"`
	TrustProviders         int `json:"trustProviders"`
	TrustInactive          int `json:"trustInactive"`
	CredentialProviders    int `json:"credentialProviders"`
	CredentialInactive     int `json:"credentialInactive"`
	AccessPolicies         int `json:"accessPolicies"`
	AccessPoliciesInactive int `json:"accessPoliciesInactive"`
	AccessConditions       int `json:"accessConditions"`
	ConditionInactive      int `json:"conditionInactive"`
}

type RecentEventStats struct {
	Last6h               EventSummary          `json:"last6h"`
	TimeSeries           []EventTimePoint      `json:"timeSeries"`
	AccessDecisionSeries []AccessDecisionPoint `json:"accessDecisionSeries"`
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

type AccessDecisionPoint struct {
	Timestamp  string `json:"timestamp"`
	Authorized int    `json:"authorized"`
	Denied     int    `json:"denied"`
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

type AttestationResult struct {
	Result string `json:"result"`
	Count  int    `json:"count"`
}

type DashboardService struct {
	cwRepo    repository.ClientWorkloadRepository
	swRepo    repository.ServerWorkloadRepository
	tpRepo    repository.TrustProviderRepository
	cpRepo    repository.CredentialProviderRepository
	apRepo    repository.AccessPolicyRepository
	acRepo    repository.AccessConditionRepository
	eventRepo repository.EventRepository
}

func NewDashboardService(
	cw repository.ClientWorkloadRepository,
	sw repository.ServerWorkloadRepository,
	tp repository.TrustProviderRepository,
	cp repository.CredentialProviderRepository,
	ap repository.AccessPolicyRepository,
	ac repository.AccessConditionRepository,
	ev repository.EventRepository,
) *DashboardService {
	return &DashboardService{
		cwRepo: cw, swRepo: sw, tpRepo: tp, cpRepo: cp, apRepo: ap, acRepo: ac, eventRepo: ev,
	}
}

func (s *DashboardService) GetStats(ctx context.Context, tenantID uuid.UUID, timeRange string) (*DashboardStats, error) {
	opts := domain.ListOptions{Page: 1, PerPage: 1}

	_, cwCount, _ := s.cwRepo.List(ctx, tenantID, opts)
	_, swCount, _ := s.swRepo.List(ctx, tenantID, opts)
	_, tpCount, _ := s.tpRepo.List(ctx, tenantID, opts)
	_, cpCount, _ := s.cpRepo.List(ctx, tenantID, opts)
	_, apCount, _ := s.apRepo.List(ctx, tenantID, opts)
	_, acCount, _ := s.acRepo.List(ctx, tenantID, opts)

	// Get inactive counts
	cwInactive := s.cwRepo.CountInactive(ctx, tenantID)
	swInactive := s.swRepo.CountInactive(ctx, tenantID)
	tpInactive := s.tpRepo.CountInactive(ctx, tenantID)
	cpInactive := s.cpRepo.CountInactive(ctx, tenantID)
	apInactive := s.apRepo.CountInactive(ctx, tenantID)
	acInactive := s.acRepo.CountInactive(ctx, tenantID)

	stats := &DashboardStats{
		Counts: EntityCounts{
			ClientWorkloads:        cwCount,
			ClientInactive:         cwInactive,
			ServerWorkloads:        swCount,
			ServerInactive:         swInactive,
			TrustProviders:         tpCount,
			TrustInactive:          tpInactive,
			CredentialProviders:    cpCount,
			CredentialInactive:     cpInactive,
			AccessPolicies:         apCount,
			AccessPoliciesInactive: apInactive,
			AccessConditions:       acCount,
			ConditionInactive:      acInactive,
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
	}

	if s.eventRepo == nil {
		return stats, nil
	}

	eventData, err := s.eventRepo.DashboardEventStats(ctx, tenantID, timeRange)
	if err != nil || eventData == nil {
		return stats, nil
	}

	stats.RecentEvents.Last6h = EventSummary{
		Total:      eventData.TotalLast6h,
		Authorized: eventData.AuthorizedLast6h,
		Denied:     eventData.DeniedLast6h,
	}
	for _, pt := range eventData.TimeSeries {
		stats.RecentEvents.TimeSeries = append(stats.RecentEvents.TimeSeries, EventTimePoint{
			Timestamp: pt.Timestamp.Format("2006-01-02T15:04:05Z"),
			Info:      pt.Info,
			Warning:   pt.Warning,
			Error:     pt.Error,
		})
	}
	for _, pt := range eventData.AccessDecisionSeries {
		stats.RecentEvents.AccessDecisionSeries = append(stats.RecentEvents.AccessDecisionSeries, AccessDecisionPoint{
			Timestamp:  pt.Timestamp.Format("2006-01-02T15:04:05Z"),
			Authorized: pt.Authorized,
			Denied:     pt.Denied,
		})
	}
	for _, wc := range eventData.TopClients {
		stats.TopClientWorkloads = append(stats.TopClientWorkloads, TopWorkload{
			ID: wc.ID, Name: wc.Name, Connections: wc.Count,
		})
	}
	for _, wc := range eventData.TopServers {
		stats.TopServerWorkloads = append(stats.TopServerWorkloads, TopWorkload{
			ID: wc.ID, Name: wc.Name, Connections: wc.Count,
		})
	}
	for _, cu := range eventData.CredentialUsage {
		stats.CredentialUsageByType = append(stats.CredentialUsageByType, CredentialUsage{
			Type: cu.Type, Count: cu.Count,
		})
	}
	for _, pc := range eventData.ProtocolUsage {
		stats.ConnectionsByProtocol = append(stats.ConnectionsByProtocol, ProtocolCount{
			Protocol: pc.Type, Count: pc.Count,
		})
	}
	for _, pf := range eventData.PolicyFailures {
		stats.MostConditionFailures = append(stats.MostConditionFailures, PolicyFailures{
			PolicyID: pf.PolicyID, PolicyName: pf.PolicyName, Failures: pf.Count,
		})
	}
	for _, ar := range eventData.AttestationResults {
		stats.AttestationResults = append(stats.AttestationResults, AttestationResult{
			Result: ar.Type, Count: ar.Count,
		})
	}

	return stats, nil
}
