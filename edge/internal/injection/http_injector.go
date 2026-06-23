/*
http_injector.go implements individual HTTP credential injection methods.

Each function modifies the request in-place to add authentication. Credential
values are never logged anywhere in this package.
*/
package injection

import (
	"encoding/base64"
	"net/http"
)

func injectBearer(req *http.Request, cred *Credential) {
	req.Header.Set("Authorization", "Bearer "+cred.Value)
}

func injectBasic(req *http.Request, cred *Credential) {
	encoded := base64.StdEncoding.EncodeToString([]byte(cred.Username + ":" + cred.Password))
	req.Header.Set("Authorization", "Basic "+encoded)
}

func injectHeader(req *http.Request, config *InjectionConfig, cred *Credential) {
	req.Header.Set(config.HeaderName, config.Prefix+cred.Value)
}

func injectQueryParam(req *http.Request, config *InjectionConfig, cred *Credential) {
	q := req.URL.Query()
	q.Set(config.QueryParam, cred.Value)
	req.URL.RawQuery = q.Encode()
}
