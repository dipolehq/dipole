/*
Package identity gathers workload identity evidence from the local environment.

EvidenceGatherer is called on every intercepted request to collect fresh
evidence (K8s SA token, AWS IMDS/STS, Azure IMDS, GCP metadata, env vars,
source IP). Sub-gatherers that fail (e.g., not in K8s, not on AWS) are
skipped gracefully — we send whatever evidence is available.
*/
package identity

import (
	"log/slog"
	"os"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type EvidenceGatherer struct {
	logger  *slog.Logger
	awsIMDS *awsIMDSGatherer
	awsSTS  *awsSTSGatherer
	azure   *azureGatherer
	gcp     *gcpGatherer
}

func NewEvidenceGatherer(logger *slog.Logger) *EvidenceGatherer {
	imds := newIMDSClient()
	tenantID := os.Getenv("DIPOLE_TENANT_ID")
	return &EvidenceGatherer{
		logger:  logger,
		awsIMDS: newAWSIMDSGatherer(imds),
		awsSTS:  newAWSSTSGatherer(imds),
		azure:   newAzureGatherer(),
		gcp:     newGCPGatherer(tenantID),
	}
}

func (g *EvidenceGatherer) Gather(sourceIP string) *pb.ClientEvidence {
	ev := &pb.ClientEvidence{
		SourceIp:       sourceIP,
		Hostname:       hostname(),
		AembitClientId: os.Getenv("CLIENT_WORKLOAD_ID"),
		Environment:    gatherEnv(),
	}

	if k8s := gatherK8s(g.logger); k8s != nil {
		ev.K8S = k8s
	}

	if aws := g.gatherAWS(); aws != nil {
		ev.Aws = aws
	}

	if azure := g.gatherAzure(); azure != nil {
		ev.Azure = azure
	}

	if gcp := g.gatherGCP(); gcp != nil {
		ev.Gcp = gcp
	}

	return ev
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
