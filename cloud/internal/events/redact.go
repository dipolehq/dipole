/*
redact.go strips sensitive HTTP headers from workload event payloads before
database persistence.

Edge proxies stream raw request/response metadata including Authorization,
Cookie, Set-Cookie, and X-API-Key headers. These must never be stored in
cleartext. This redaction runs inside IngestFromProxy before the batch is
written, ensuring no credential material reaches the workload_events table
regardless of what the proxy sends.
*/
package events

import (
	"encoding/json"
	"strings"
)

func redactPayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return payload
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	modified := false

	if app, ok := data["application"].(map[string]any); ok {
		if http, ok := app["http"].(map[string]any); ok {
			if headers, ok := http["headers"].([]any); ok {
				http["headers"] = redactHeaders(headers)
				modified = true
			}
		}
		if mcp, ok := app["mcp"].(map[string]any); ok {
			if headers, ok := mcp["headers"].([]any); ok {
				mcp["headers"] = redactHeaders(headers)
				modified = true
			}
		}
	}

	if !modified {
		return payload
	}

	result, err := json.Marshal(data)
	if err != nil {
		return payload
	}
	return result
}

func redactHeaders(headers []any) []any {
	redacted := make([]any, 0, len(headers))
	for _, h := range headers {
		header, ok := h.(map[string]any)
		if !ok {
			redacted = append(redacted, h)
			continue
		}

		shouldRedact := false
		for key := range header {
			lower := strings.ToLower(key)
			if lower == "authorization" || lower == "set-cookie" || lower == "cookie" || lower == "x-api-key" {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			redacted = append(redacted, map[string]any{"[REDACTED]": "[REDACTED]"})
		} else {
			redacted = append(redacted, header)
		}
	}
	return redacted
}
