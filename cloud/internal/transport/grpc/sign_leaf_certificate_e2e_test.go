// sign_leaf_certificate_e2e_test.go exercises the real signing flow end to end over a bufconn gRPC server backed by the actual pki.TenantCAService, verifying the returned leaf certificate chains to the tenant's Root CA.
package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type inMemoryProxyRepo struct {
	mu      sync.Mutex
	proxies map[string]*domain.AgentProxy
}

func newInMemoryProxyRepo() *inMemoryProxyRepo {
	return &inMemoryProxyRepo{proxies: make(map[string]*domain.AgentProxy)}
}

func (r *inMemoryProxyRepo) registerWithToken(tenantID, proxyID uuid.UUID, sessionToken string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.proxies[hashSessionToken(sessionToken)] = &domain.AgentProxy{ID: proxyID, TenantID: tenantID}
}

func (r *inMemoryProxyRepo) GetBySessionTokenHash(_ context.Context, tokenHash string) (*domain.AgentProxy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.proxies[tokenHash], nil
}

func generateTestSessionToken(t *testing.T) string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("generate session token: %v", err)
	}
	return hex.EncodeToString(raw)
}

func withSessionToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

type inMemoryTenantCARepo struct {
	mu       sync.Mutex
	certPEM  string
	keyEnc   []byte
	lifetime int
}

func (r *inMemoryTenantCARepo) GetOrCreateCA(_ context.Context, _ uuid.UUID, certPEM string, keyEncrypted []byte) (string, []byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.certPEM != "" {
		return r.certPEM, r.keyEnc, nil
	}
	r.certPEM = certPEM
	r.keyEnc = keyEncrypted
	return certPEM, keyEncrypted, nil
}

func (r *inMemoryTenantCARepo) GetCA(_ context.Context, _ uuid.UUID) (string, []byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.certPEM == "" {
		return "", nil, nil
	}
	return r.certPEM, r.keyEnc, nil
}

func (r *inMemoryTenantCARepo) HasCA(_ context.Context, _ uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.certPEM != "", nil
}

func (r *inMemoryTenantCARepo) GetLeafLifetime(_ context.Context, _ uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lifetime == 0 {
		return 86400, nil
	}
	return r.lifetime, nil
}

func (r *inMemoryTenantCARepo) SetLeafLifetime(_ context.Context, _ uuid.UUID, lifetimeSeconds int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lifetime = lifetimeSeconds
	return nil
}

func buildProxyCSR(t *testing.T, hostname string) (*ecdsa.PrivateKey, []byte) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ephemeral leaf key: %v", err)
	}
	template := &x509.CertificateRequest{DNSNames: []string{hostname}}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}
	return key, csrDER
}

func startBufconnServer(t *testing.T, srv *PolicyEvaluationServer, proxyRepo SessionProxyAuthenticator) pb.PolicyEvaluationServiceClient {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(SessionAuthInterceptor(proxyRepo)),
		grpc.ChainStreamInterceptor(SessionAuthStreamInterceptor(proxyRepo)),
	)
	pb.RegisterPolicyEvaluationServiceServer(grpcServer, srv)
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(grpcServer.Stop)

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return pb.NewPolicyEvaluationServiceClient(conn)
}

func TestE2E_SignLeafCertificate_RealSigningFlow(t *testing.T) {
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatalf("generate encryption key: %v", err)
	}

	repo := &inMemoryTenantCARepo{}
	caService := pki.NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	proxyID := uuid.New()
	const hostname = "api.stripe.com"

	tenantCACertPEM, _, err := caService.GetOrCreateTenantCA(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetOrCreateTenantCA: %v", err)
	}

	hostAuth := &mockHostnameAuthorizer{authorized: map[string]bool{hostname: true}}

	server := &PolicyEvaluationServer{
		leafSigner:    caService,
		hostnameAuthz: hostAuth,
		logger:        newTestLogger(),
	}

	sessionToken := generateTestSessionToken(t)
	proxyRepo := newInMemoryProxyRepo()
	proxyRepo.registerWithToken(tenantID, proxyID, sessionToken)

	client := startBufconnServer(t, server, proxyRepo)

	leafKey, csrDER := buildProxyCSR(t, hostname)

	ctx, cancel := context.WithTimeout(withSessionToken(context.Background(), sessionToken), 5*time.Second)
	defer cancel()

	resp, err := client.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   csrDER,
		Hostname: hostname,
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("SignLeafCertificate rpc: %v", err)
	}
	if !resp.Success {
		t.Fatalf("signing denied: %s", resp.ErrorMessage)
	}

	leafCert, err := x509.ParseCertificate(resp.CertDer)
	if err != nil {
		t.Fatalf("parse returned leaf cert: %v", err)
	}

	caBlock, _ := pem.Decode([]byte(tenantCACertPEM))
	if caBlock == nil {
		t.Fatal("failed to decode tenant CA cert PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("parse tenant CA cert: %v", err)
	}

	if err := leafCert.CheckSignatureFrom(caCert); err != nil {
		t.Fatalf("leaf certificate does not chain to tenant Root CA: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := leafCert.Verify(x509.VerifyOptions{
		DNSName:   hostname,
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Fatalf("leaf certificate failed full chain verification for %s: %v", hostname, err)
	}

	if leafCert.IsCA {
		t.Error("leaf certificate must not have IsCA=true")
	}
	if len(leafCert.DNSNames) != 1 || leafCert.DNSNames[0] != hostname {
		t.Errorf("leaf DNSNames = %v, want [%s]", leafCert.DNSNames, hostname)
	}

	leafPub, ok := leafCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("leaf public key is not ECDSA")
	}
	if leafPub.X.Cmp(leafKey.PublicKey.X) != 0 || leafPub.Y.Cmp(leafKey.PublicKey.Y) != 0 {
		t.Fatal("leaf certificate's public key does not match the proxy's ephemeral private key")
	}

	if resp.NotAfter <= time.Now().Unix() {
		t.Error("not_after is not in the future")
	}
}

func TestE2E_SignLeafCertificate_CACreatedOnlyAtRegistration(t *testing.T) {
	encKey := make([]byte, 32)
	rand.Read(encKey)
	repo := &inMemoryTenantCARepo{}
	caService := pki.NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	proxyID := uuid.New()
	const hostname = "api.stripe.com"

	hostAuth := &mockHostnameAuthorizer{authorized: map[string]bool{hostname: true}}

	server := &PolicyEvaluationServer{
		leafSigner:    caService,
		hostnameAuthz: hostAuth,
		logger:        newTestLogger(),
	}

	sessionToken := generateTestSessionToken(t)
	proxyRepo := newInMemoryProxyRepo()
	proxyRepo.registerWithToken(tenantID, proxyID, sessionToken)

	client := startBufconnServer(t, server, proxyRepo)

	_, csrDER := buildProxyCSR(t, hostname)
	ctx, cancel := context.WithTimeout(withSessionToken(context.Background(), sessionToken), 5*time.Second)
	defer cancel()

	resp, err := client.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   csrDER,
		Hostname: hostname,
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("SignLeafCertificate rpc: %v", err)
	}
	if resp.Success {
		t.Fatal("signing succeeded without a tenant CA ever being created (lazy CA generation must not happen)")
	}
	has, _ := repo.HasCA(context.Background(), tenantID)
	if has {
		t.Fatal("a CA was created as a side effect of a sign request")
	}
}

func BenchmarkSignLeafCertificate_RoundTrip(b *testing.B) {
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		b.Fatalf("generate encryption key: %v", err)
	}

	repo := &inMemoryTenantCARepo{}
	caService := pki.NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	proxyID := uuid.New()
	const hostname = "api.stripe.com"

	if _, _, err := caService.GetOrCreateTenantCA(context.Background(), tenantID); err != nil {
		b.Fatalf("GetOrCreateTenantCA: %v", err)
	}

	hostAuth := &mockHostnameAuthorizer{authorized: map[string]bool{hostname: true}}

	server := &PolicyEvaluationServer{
		leafSigner:    caService,
		hostnameAuthz: hostAuth,
		logger:        newTestLogger(),
	}

	sessionToken := make([]byte, 32)
	if _, err := rand.Read(sessionToken); err != nil {
		b.Fatalf("generate session token: %v", err)
	}
	sessionTokenHex := hex.EncodeToString(sessionToken)
	proxyRepo := newInMemoryProxyRepo()
	proxyRepo.registerWithToken(tenantID, proxyID, sessionTokenHex)

	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(SessionAuthInterceptor(proxyRepo)),
		grpc.ChainStreamInterceptor(SessionAuthStreamInterceptor(proxyRepo)),
	)
	pb.RegisterPolicyEvaluationServiceServer(grpcServer, server)
	go func() { _ = grpcServer.Serve(lis) }()
	b.Cleanup(grpcServer.Stop)

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Fatalf("dial bufconn: %v", err)
	}
	b.Cleanup(func() { conn.Close() })
	client := pb.NewPolicyEvaluationServiceClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			b.Fatalf("generate ephemeral key: %v", err)
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{hostname}}, key)
		if err != nil {
			b.Fatalf("create CSR: %v", err)
		}

		ctx, cancel := context.WithTimeout(withSessionToken(context.Background(), sessionTokenHex), 5*time.Second)
		resp, err := client.SignLeafCertificate(ctx, &pb.SignLeafRequest{
			CsrDer:   csrDER,
			Hostname: hostname,
			ProxyId:  proxyID.String(),
			TenantId: tenantID.String(),
		})
		cancel()
		if err != nil {
			b.Fatalf("SignLeafCertificate: %v", err)
		}
		if !resp.Success {
			b.Fatalf("signing denied: %s", resp.ErrorMessage)
		}
	}
}
