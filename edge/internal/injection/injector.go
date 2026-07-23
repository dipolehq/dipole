/*
Package injection provides credential injection into HTTP requests.

Credential holds the value(s) to inject (from Cloud PDP). InjectionConfig
describes how to inject based on the Server Workload's auth method.
InjectCredential dispatches to the appropriate injection function.
*/
package injection

import (
	"fmt"
	"net/http"
)

type Credential struct {
	Type               string
	Value              string
	Username           string
	Password           string
	AwsAccessKeyId     string
	AwsSecretAccessKey string
	AwsSessionToken    string
}

type InjectionConfig struct {
	Method     string
	HeaderName string
	Prefix     string
	QueryParam string
}

type Injector interface {
	Inject(req *http.Request, cred *Credential, config *InjectionConfig) error
}

func InjectCredential(req *http.Request, cred *Credential, config *InjectionConfig) error {
	switch config.Method {
	case "http_bearer":
		injectBearer(req, cred)
	case "http_basic":
		injectBasic(req, cred)
	case "http_header", "api_key_header":
		injectHeader(req, config, cred)
	case "api_key_query":
		injectQueryParam(req, config, cred)
	default:
		return fmt.Errorf("unsupported injection method: %s", config.Method)
	}
	return nil
}
