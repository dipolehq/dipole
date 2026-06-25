/*
env.go reads environment variables relevant to workload identification.

GatherEnv returns a map of env vars that are sent as evidence to Cloud for
client workload identification and policy matching.
*/
package identity

import "os"

var envKeys = []string{
	"CLIENT_WORKLOAD_ID",
	"K8S_POD_NAME",
	"K8S_NAMESPACE",
	"KUBERNETES_PROVIDER_ID",
	"DIPOLE_RESOURCE_SET_ID",
}

func gatherEnv() map[string]string {
	env := make(map[string]string)
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			env[key] = v
		}
	}
	return env
}
