/*
kubernetes.go reads Kubernetes identity evidence from the local environment.

GatherK8s reads the projected service account token from the filesystem on
every call (tokens rotate), decodes the JWT payload to extract the service
account name and UID. Also reads K8S_POD_NAME and K8S_NAMESPACE env vars.
Returns nil if not in a K8s environment.
*/
package identity

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

const saTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func gatherK8s(logger *slog.Logger) *pb.K8SEvidence {
	token, err := os.ReadFile(saTokenPath)
	if err != nil {
		logger.Debug("not in kubernetes: SA token not found", "path", saTokenPath)
		return nil
	}

	tokenStr := strings.TrimSpace(string(token))

	var saName, saUID string
	claims, err := decodeJWTPayload(tokenStr)
	if err != nil {
		logger.Warn("failed to decode SA token JWT", "error", err)
	} else {
		saName, saUID = extractServiceAccount(claims)
	}

	return &pb.K8SEvidence{
		ServiceAccountToken: tokenStr,
		Namespace:           os.Getenv("K8S_NAMESPACE"),
		PodName:             os.Getenv("K8S_POD_NAME"),
		ServiceAccountName:  saName,
		ServiceAccountUid:   saUID,
	}
}

func decodeJWTPayload(token string) (map[string]interface{}, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}

	// JWT uses base64url encoding without padding
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	return claims, nil
}

func extractServiceAccount(claims map[string]interface{}) (name, uid string) {
	k8sClaims, ok := claims["kubernetes.io"].(map[string]interface{})
	if !ok {
		return "", ""
	}

	sa, ok := k8sClaims["serviceaccount"].(map[string]interface{})
	if !ok {
		return "", ""
	}

	name, _ = sa["name"].(string)
	uid, _ = sa["uid"].(string)
	return name, uid
}
