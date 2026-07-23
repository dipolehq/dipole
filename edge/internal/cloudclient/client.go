/*
Package cloudclient provides the gRPC client for connecting Agent Proxy to
Dipole Cloud for policy evaluation, registration, health reporting, and
event ingestion.

CloudClient maintains a persistent gRPC connection and exposes Evaluate,
Register, ReportHealth, and IngestEvents to communicate with the Cloud
control plane.
*/
package cloudclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type CloudClient struct {
	conn          *grpc.ClientConn
	client        pb.PolicyEvaluationServiceClient
	tenantID      string
	proxyID       string // set after successful registration
	resourceSetID string // from config, set via setter
	logger        *slog.Logger
	connected     atomic.Bool
}

func NewCloudClient(endpoint string, tenantID string, logger *slog.Logger) (*CloudClient, error) {
	kaParams := keepalive.ClientParameters{
		Time:                60 * time.Second,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaParams),
	)
	if err != nil {
		return nil, fmt.Errorf("dial cloud %s: %w", endpoint, err)
	}

	c := &CloudClient{
		conn:     conn,
		client:   pb.NewPolicyEvaluationServiceClient(conn),
		tenantID: tenantID,
		logger:   logger,
	}
	c.connected.Store(true)
	return c, nil
}

// SetResourceSetID sets the resource set ID (called from main.go with config value).
func (c *CloudClient) SetResourceSetID(id string) {
	c.resourceSetID = id
}

// ProxyID returns the proxy ID obtained from registration. Empty if not yet registered.
func (c *CloudClient) ProxyID() string {
	return c.proxyID
}

type RegisterResult struct {
	ProxyID   string
	CACertPEM string
	CAKeyPEM  string
}

// Register calls the Cloud Register RPC with the given token and version.
// On success, it stores the returned proxyID internally and returns a RegisterResult
// containing the proxy ID and optional tenant CA material.
func (c *CloudClient) Register(ctx context.Context, token, version string) (*RegisterResult, error) {
	req := &pb.RegisterRequest{
		TenantId:          c.tenantID,
		RegistrationToken: token,
		ProxyVersion:      version,
	}

	resp, err := c.client.Register(ctx, req)
	if err != nil {
		c.connected.Store(false)
		return nil, fmt.Errorf("register rpc: %w", err)
	}

	c.connected.Store(true)

	if !resp.Success {
		return nil, fmt.Errorf("registration denied: %s", resp.ErrorMessage)
	}

	c.proxyID = resp.ProxyId
	return &RegisterResult{
		ProxyID:   resp.ProxyId,
		CACertPEM: resp.CaCertPem,
		CAKeyPEM:  resp.CaKeyPem,
	}, nil
}

// ReportHealth sends a health report to Cloud indicating this proxy is healthy.
func (c *CloudClient) ReportHealth(ctx context.Context, activeConns int32) error {
	req := &pb.HealthReport{
		ComponentId:       c.proxyID,
		TenantId:          c.tenantID,
		Status:            "Healthy",
		ActiveConnections: activeConns,
		Timestamp:         time.Now().Unix(),
	}

	resp, err := c.client.ReportHealth(ctx, req)
	if err != nil {
		c.connected.Store(false)
		return fmt.Errorf("health rpc: %w", err)
	}

	c.connected.Store(true)

	if !resp.Acknowledged {
		return fmt.Errorf("health report not acknowledged")
	}

	return nil
}

// IngestEvents opens a client-streaming RPC and sends a batch of workload events to Cloud.
func (c *CloudClient) IngestEvents(ctx context.Context, events []*pb.WorkloadEvent) error {
	stream, err := c.client.IngestEvents(ctx)
	if err != nil {
		return fmt.Errorf("open event stream: %w", err)
	}

	batch := &pb.WorkloadEventBatch{
		TenantId: c.tenantID,
		Events:   events,
	}

	if err := stream.Send(batch); err != nil {
		return fmt.Errorf("send event batch: %w", err)
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close event stream: %w", err)
	}

	return nil
}

func (c *CloudClient) Evaluate(ctx context.Context, evidence *pb.ClientEvidence, host string, port int) (*EvaluationResult, error) {
	req := &pb.PolicyEvaluationRequest{
		TenantId:      c.tenantID,
		ProxyId:       c.proxyID,
		ResourceSetId: c.resourceSetID,
		Client:        evidence,
		Server:        &pb.ServerTarget{Host: host, Port: int32(port)},
	}

	resp, err := c.client.Evaluate(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			c.connected.Store(false)
			c.logger.Error("cloud unreachable", "endpoint", c.conn.Target(), "error", err)
		}
		return nil, fmt.Errorf("evaluate rpc: %w", err)
	}

	c.connected.Store(true)
	return mapResult(resp), nil
}

func (c *CloudClient) IsConnected() bool {
	return c.connected.Load()
}

func (c *CloudClient) Close() error {
	return c.conn.Close()
}

type EvaluationResult struct {
	Allowed            bool
	DenyReason         string
	CredentialType     string
	CredentialValue    string
	Username           string
	Password           string
	AwsAccessKeyId     string
	AwsSecretAccessKey string
	AwsSessionToken    string
	ExpiresAt          int64
	InjectionMethod    string
	InjectionHeader    string
	InjectionPrefix    string
	InjectionQuery     string
	PolicyID           string
	ClientWorkloadID   string
	ServerWorkloadID   string
}

func mapResult(resp *pb.PolicyEvaluationResponse) *EvaluationResult {
	r := &EvaluationResult{
		Allowed:          resp.Allowed,
		DenyReason:       resp.DenyReason,
		PolicyID:         resp.PolicyId,
		ClientWorkloadID: resp.ClientWorkloadId,
		ServerWorkloadID: resp.ServerWorkloadId,
	}

	if resp.Credential != nil {
		r.CredentialType = resp.Credential.Type
		r.CredentialValue = resp.Credential.Value
		r.Username = resp.Credential.Username
		r.Password = resp.Credential.Password
		r.AwsAccessKeyId = resp.Credential.AwsAccessKeyId
		r.AwsSecretAccessKey = resp.Credential.AwsSecretAccessKey
		r.AwsSessionToken = resp.Credential.AwsSessionToken
		r.ExpiresAt = resp.Credential.ExpiresAt
	}

	if resp.Injection != nil {
		r.InjectionMethod = resp.Injection.Method
		r.InjectionHeader = resp.Injection.HeaderName
		r.InjectionPrefix = resp.Injection.Prefix
		r.InjectionQuery = resp.Injection.QueryParam
	}

	return r
}
