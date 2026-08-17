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
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dipolehq/dipole/edge/internal/metrics"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type CloudClient struct {
	conn          *grpc.ClientConn
	client        pb.PolicyEvaluationServiceClient
	tenantID      string
	proxyID       string
	resourceSetID string
	logger        *slog.Logger
	connected     atomic.Bool
	// sessionToken is the bearer token issued by Register and required by
	// every other RPC on this connection (see sessionAuthUnaryInterceptor).
	// Stored in an atomic.Value since Register runs once at startup but
	// ReportHealth/Evaluate/SignLeaf may be read concurrently from other
	// goroutines afterward.
	sessionToken atomic.Value
}

func NewCloudClient(endpoint string, tenantID string, trustPath string, grpcInsecure bool, logger *slog.Logger) (*CloudClient, error) {
	kaParams := keepalive.ClientParameters{
		Time:                60 * time.Second,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}

	creds, err := buildTransportCredentials(trustPath, grpcInsecure)
	if err != nil {
		return nil, fmt.Errorf("build transport credentials: %w", err)
	}

	c := &CloudClient{
		tenantID: tenantID,
		logger:   logger,
	}
	c.sessionToken.Store("")

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kaParams),
		grpc.WithChainUnaryInterceptor(c.sessionAuthUnaryInterceptor),
		grpc.WithChainStreamInterceptor(c.sessionAuthStreamInterceptor),
	}

	if dialer := buildProxyDialer(); dialer != nil {
		dialOpts = append(dialOpts, grpc.WithContextDialer(dialer))
	}

	conn, err := grpc.NewClient(endpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial cloud %s: %w", endpoint, err)
	}

	c.conn = conn
	c.client = pb.NewPolicyEvaluationServiceClient(conn)
	c.connected.Store(true)
	metrics.GRPCConnectionStatus.Set(1)
	return c, nil
}

// sessionAuthUnaryInterceptor attaches the bearer session token (if one has
// been issued) to every outgoing unary call as gRPC metadata. Register is
// unaffected in practice since it runs before any token exists — the
// metadata is simply empty, which the server-side interceptor also exempts
// Register from requiring.
func (c *CloudClient) sessionAuthUnaryInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(c.attachSessionToken(ctx), method, req, reply, cc, opts...)
}

// sessionAuthStreamInterceptor is the streaming counterpart, needed for
// IngestEvents.
func (c *CloudClient) sessionAuthStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(c.attachSessionToken(ctx), desc, cc, method, opts...)
}

func (c *CloudClient) attachSessionToken(ctx context.Context) context.Context {
	token, _ := c.sessionToken.Load().(string)
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

func buildTransportCredentials(trustPath string, grpcInsecure bool) (credentials.TransportCredentials, error) {
	if grpcInsecure {
		return insecure.NewCredentials(), nil
	}
	var certPool *x509.CertPool
	if trustPath != "" {
		pem, err := os.ReadFile(trustPath)
		if err != nil {
			return nil, fmt.Errorf("read CA cert %s: %w", trustPath, err)
		}
		certPool = x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse CA cert from %s", trustPath)
		}
	}
	return credentials.NewTLS(&tls.Config{RootCAs: certPool}), nil
}

func buildProxyDialer() func(ctx context.Context, addr string) (net.Conn, error) {
	proxyURL := os.Getenv("HTTPS_PROXY")
	if proxyURL == "" {
		proxyURL = os.Getenv("HTTP_PROXY")
	}
	if proxyURL == "" {
		return nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil
	}

	return func(ctx context.Context, addr string) (net.Conn, error) {
		if isNoProxy(addr) {
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
		}
		return dialViaHTTPConnect(ctx, parsed, addr)
	}
}

func dialViaHTTPConnect(ctx context.Context, proxyURL *url.URL, target string) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		if proxyURL.Scheme == "https" {
			proxyAddr += ":443"
		} else {
			proxyAddr += ":8080"
		}
	}

	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("dial proxy %s: %w", proxyAddr, err)
	}

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", target, target)
	if proxyURL.User != nil {
		auth := proxyURL.User.String()
		connectReq += "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(auth)) + "\r\n"
	}
	connectReq += "\r\n"

	if _, err := conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send CONNECT to proxy: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read proxy CONNECT response: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT returned %d", resp.StatusCode)
	}

	return conn, nil
}

func isNoProxy(addr string) bool {
	noProxy := os.Getenv("NO_PROXY")
	if noProxy == "" {
		return false
	}

	host, _, _ := net.SplitHostPort(addr)
	if host == "" {
		host = addr
	}

	for _, pattern := range strings.Split(noProxy, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == "*" {
			return true
		}
		if strings.HasPrefix(pattern, ".") {
			if strings.HasSuffix(host, pattern) || host == pattern[1:] {
				return true
			}
		} else if host == pattern {
			return true
		}
	}
	return false
}

func (c *CloudClient) SetResourceSetID(id string) {
	c.resourceSetID = id
}

func (c *CloudClient) ResourceSetID() string {
	return c.resourceSetID
}

func (c *CloudClient) ProxyID() string {
	return c.proxyID
}

type RegisterResult struct {
	ProxyID   string
	CACertPEM string
}

func (c *CloudClient) Register(ctx context.Context, token, version string) (*RegisterResult, error) {
	req := &pb.RegisterRequest{
		TenantId:          c.tenantID,
		RegistrationToken: token,
		ProxyVersion:      version,
	}

	resp, err := c.client.Register(ctx, req)
	if err != nil {
		c.connected.Store(false)
		metrics.GRPCConnectionStatus.Set(0)
		return nil, fmt.Errorf("register rpc: %w", err)
	}

	c.connected.Store(true)
	metrics.GRPCConnectionStatus.Set(1)

	if !resp.Success {
		return nil, fmt.Errorf("registration denied: %s", resp.ErrorMessage)
	}

	if resp.SessionToken == "" {
		return nil, fmt.Errorf("registration succeeded but cloud did not issue a session token")
	}

	c.proxyID = resp.ProxyId
	c.sessionToken.Store(resp.SessionToken)
	return &RegisterResult{
		ProxyID:   resp.ProxyId,
		CACertPEM: resp.CaCertPem,
	}, nil
}

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
		metrics.GRPCConnectionStatus.Set(0)
		return fmt.Errorf("health rpc: %w", err)
	}

	c.connected.Store(true)
	metrics.GRPCConnectionStatus.Set(1)

	if !resp.Acknowledged {
		return fmt.Errorf("health report not acknowledged")
	}

	return nil
}

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
			metrics.GRPCConnectionStatus.Set(0)
			c.logger.Error("cloud unreachable", "endpoint", c.conn.Target(), "error", err)
		}
		return nil, fmt.Errorf("evaluate rpc: %w", err)
	}

	c.connected.Store(true)
	metrics.GRPCConnectionStatus.Set(1)
	return mapResult(resp), nil
}

// SignLeaf implements dipoletls.LeafSigner: it sends a proxy-generated CSR
// to Cloud's SignLeafCertificate RPC and returns the signed leaf
// certificate. The tenant's Root CA private key stays in Cloud — only the
// signed certificate crosses the wire.
func (c *CloudClient) SignLeaf(ctx context.Context, csrDER []byte, hostname string) ([]byte, time.Time, error) {
	req := &pb.SignLeafRequest{
		CsrDer:   csrDER,
		Hostname: hostname,
		ProxyId:  c.proxyID,
		TenantId: c.tenantID,
	}

	resp, err := c.client.SignLeafCertificate(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			c.connected.Store(false)
			metrics.GRPCConnectionStatus.Set(0)
		}
		return nil, time.Time{}, fmt.Errorf("sign leaf certificate rpc: %w", err)
	}

	if !resp.Success {
		return nil, time.Time{}, fmt.Errorf("leaf certificate signing denied: %s", resp.ErrorMessage)
	}

	return resp.CertDer, time.Unix(resp.NotAfter, 0), nil
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
