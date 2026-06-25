/*
kubernetes.go reads Kubernetes identity evidence from the local environment.

GatherK8s reads the projected service account token from the filesystem on
every call (tokens rotate). Also reads K8S_POD_NAME, K8S_NAMESPACE, and
KUBERNETES_PROVIDER_ID env vars. Returns nil if not in a K8s environment.
*/
package identity

import (
	"log/slog"
	"os"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

const saTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func gatherK8s(logger *slog.Logger) *pb.K8SEvidence {
	token, err := os.ReadFile(saTokenPath)
	if err != nil {
		logger.Debug("not in kubernetes: SA token not found", "path", saTokenPath)
		return nil
	}

	return &pb.K8SEvidence{
		ServiceAccountToken: string(token),
		Namespace:           os.Getenv("K8S_NAMESPACE"),
		PodName:             os.Getenv("K8S_POD_NAME"),
		ServiceAccountName:  "",
		ServiceAccountUid:   "",
	}
}
