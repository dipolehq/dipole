/*
Package identity gathers workload identity evidence from the local environment.

EvidenceGatherer is called on every intercepted request to collect fresh
evidence (K8s SA token, env vars, source IP). Sub-gatherers that fail (e.g.,
not in K8s) are skipped gracefully — we send whatever evidence is available.
*/
package identity

import (
	"log/slog"
	"os"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type EvidenceGatherer struct {
	logger *slog.Logger
}

func NewEvidenceGatherer(logger *slog.Logger) *EvidenceGatherer {
	return &EvidenceGatherer{logger: logger}
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

	return ev
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}
