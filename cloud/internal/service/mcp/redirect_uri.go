// redirect_uri.go implements MatchRedirectURI, which does exact matching except for localhost/127.0.0.1, where ports are ignored to accommodate local dev tools using ephemeral ports.
package mcp

import (
	"net/url"
	"strings"
)

func MatchRedirectURI(provided, registered string) bool {
	if provided == registered {
		return true
	}

	pURL, err := url.Parse(provided)
	if err != nil {
		return false
	}
	rURL, err := url.Parse(registered)
	if err != nil {
		return false
	}

	if !isLocalhost(pURL.Hostname()) || !isLocalhost(rURL.Hostname()) {
		return false
	}

	return pURL.Scheme == rURL.Scheme &&
		normalizeLocalhost(pURL.Hostname()) == normalizeLocalhost(rURL.Hostname()) &&
		pURL.Path == rURL.Path
}

func MatchRedirectURIInList(provided string, registered []string) bool {
	for _, r := range registered {
		if MatchRedirectURI(provided, r) {
			return true
		}
	}
	return false
}

func isLocalhost(host string) bool {
	h := strings.ToLower(host)
	return h == "localhost" || h == "127.0.0.1" || h == "[::1]"
}

func normalizeLocalhost(host string) string {
	h := strings.ToLower(host)
	if h == "127.0.0.1" || h == "[::1]" {
		return "localhost"
	}
	return h
}
