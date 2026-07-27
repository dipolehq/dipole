/*
Package identity gathers workload identity evidence from the local environment.

EvidenceGatherer collects identity evidence (K8s SA token, AWS IMDS/STS, Azure
IMDS, GCP metadata, env vars, source IP) and caches it with a short TTL to
avoid repeated I/O on every request. Sub-gatherers that fail (e.g., not in K8s,
not on AWS) are skipped gracefully — we send whatever evidence is available.
*/
package identity

import (
	"log/slog"
	"os"
	"sync"
	"time"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

const evidenceCacheTTL = 30 * time.Second

type EvidenceGatherer struct {
	logger         *slog.Logger
	awsIMDS        *awsIMDSGatherer
	awsSTS         *awsSTSGatherer
	azure          *azureGatherer
	gcp            *gcpGatherer
	processEnabled bool

	mu     sync.RWMutex
	cached *cachedEvidence
}

type cachedEvidence struct {
	k8s       *pb.K8SEvidence
	aws       *pb.AwsEvidence
	azure     *pb.AzureEvidence
	gcp       *pb.GcpEvidence
	expiresAt time.Time
}

func NewEvidenceGatherer(logger *slog.Logger, processEnabled bool) *EvidenceGatherer {
	imds := newIMDSClient()
	tenantID := os.Getenv("DIPOLE_TENANT_ID")
	return &EvidenceGatherer{
		logger:         logger,
		awsIMDS:        newAWSIMDSGatherer(imds),
		awsSTS:         newAWSSTSGatherer(imds),
		azure:          newAzureGatherer(),
		gcp:            newGCPGatherer(tenantID),
		processEnabled: processEnabled,
	}
}

func (g *EvidenceGatherer) Gather(sourceIP string, sourcePort int) *pb.ClientEvidence {
	ev := &pb.ClientEvidence{
		SourceIp:       sourceIP,
		Hostname:       hostname(),
		DipoleClientId: os.Getenv("CLIENT_WORKLOAD_ID"),
		Environment:    gatherEnv(),
	}

	platform := g.getPlatformEvidence()
	ev.K8S = platform.k8s
	ev.Aws = platform.aws
	ev.Azure = platform.azure
	ev.Gcp = platform.gcp

	if g.processEnabled {
		if proc := gatherProcess(sourcePort); proc != nil {
			ev.Process = proc
		}
	}

	return ev
}

func (g *EvidenceGatherer) getPlatformEvidence() *cachedEvidence {
	g.mu.RLock()
	if g.cached != nil && time.Now().Before(g.cached.expiresAt) {
		c := g.cached
		g.mu.RUnlock()
		return c
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cached != nil && time.Now().Before(g.cached.expiresAt) {
		return g.cached
	}

	g.cached = &cachedEvidence{
		k8s:       g.gatherK8s(),
		aws:       g.gatherAWS(),
		azure:     g.gatherAzure(),
		gcp:       g.gatherGCP(),
		expiresAt: time.Now().Add(evidenceCacheTTL),
	}

	return g.cached
}

func (g *EvidenceGatherer) gatherK8s() *pb.K8SEvidence {
	return gatherK8s(g.logger)
}

func (g *EvidenceGatherer) gatherAWS() *pb.AwsEvidence {
	imds, _ := g.awsIMDS.Gather()
	sts, _ := g.awsSTS.Gather()

	if imds == nil && sts == nil {
		return nil
	}

	aws := &pb.AwsEvidence{}

	if imds != nil {
		aws.InstanceIdentityDocument = imds.IdentityDocument
		aws.InstanceIdentitySignature = imds.IdentitySignature
	}

	if sts != nil {
		aws.StsGetCallerIdentity = &pb.StsGetCallerIdentity{
			SignedHeaders: sts.SignedHeaders,
			Region:        sts.Region,
		}
	}

	return aws
}

func (g *EvidenceGatherer) gatherAzure() *pb.AzureEvidence {
	ev, _ := g.azure.Gather()
	if ev == nil {
		return nil
	}
	return &pb.AzureEvidence{
		Encoding:  ev.Encoding,
		Signature: ev.Signature,
		Nonce:     ev.Nonce,
	}
}

func (g *EvidenceGatherer) gatherGCP() *pb.GcpEvidence {
	ev, _ := g.gcp.Gather()
	if ev == nil {
		return nil
	}
	return &pb.GcpEvidence{
		IdentityToken: ev.IdentityToken,
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}
