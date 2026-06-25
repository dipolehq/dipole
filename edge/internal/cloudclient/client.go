/*
Package cloudclient provides the gRPC client for connecting Agent Proxy to
Dipole Cloud for policy evaluation.

CloudClient maintains a persistent gRPC connection and exposes Evaluate to
send policy evaluation requests and receive allow/deny decisions with credentials.
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
	conn      *grpc.ClientConn
	client    pb.PolicyEvaluationServiceClient
	tenantID  string
	logger    *slog.Logger
	connected atomic.Bool
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

func (c *CloudClient) Evaluate(ctx context.Context, evidence *pb.ClientEvidence, host string, port int) (*EvaluationResult, error) {
	req := &pb.PolicyEvaluationRequest{
		TenantId: c.tenantID,
		Client:   evidence,
		Server:   &pb.ServerTarget{Host: host, Port: int32(port)},
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
	Allowed          bool
	DenyReason       string
	CredentialType   string
	CredentialValue  string
	Username         string
	Password         string
	InjectionMethod  string
	InjectionHeader  string
	InjectionPrefix  string
	InjectionQuery   string
	PolicyID         string
	ClientWorkloadID string
	ServerWorkloadID string
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
	}

	if resp.Injection != nil {
		r.InjectionMethod = resp.Injection.Method
		r.InjectionHeader = resp.Injection.HeaderName
		r.InjectionPrefix = resp.Injection.Prefix
		r.InjectionQuery = resp.Injection.QueryParam
	}

	return r
}
