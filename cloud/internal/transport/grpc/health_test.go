// health_test.go tests the standard gRPC health-check service registered in StartServer: it must be reachable without a proxy session token and report SERVING.
package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func startBufconnServerWithHealth(t *testing.T, proxyRepo SessionProxyAuthenticator) (healthpb.HealthClient, *health.Server) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(SessionAuthInterceptor(proxyRepo)),
		grpc.ChainStreamInterceptor(SessionAuthStreamInterceptor(proxyRepo)),
	)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	go func() { _ = grpcServer.Serve(lis) }()
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

	return healthpb.NewHealthClient(conn), healthServer
}

func TestHealthCheck_NoSessionToken_ReportsServing(t *testing.T) {
	proxyRepo := newInMemoryProxyRepo()
	client, _ := startBufconnServerWithHealth(t, proxyRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check must not require session auth: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		t.Errorf("status = %v, want SERVING", resp.Status)
	}
}

func TestHealthCheck_ReflectsNotServingStatus(t *testing.T) {
	proxyRepo := newInMemoryProxyRepo()
	client, healthServer := startBufconnServerWithHealth(t, proxyRepo)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Errorf("status = %v, want NOT_SERVING", resp.Status)
	}
}
