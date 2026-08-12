# MCP Identity Gateway — Client Compatibility Specification

This document defines the requirements our MCP Identity Gateway MUST follow to ensure compatibility with all major MCP clients. It is derived from the MCP protocol specification (2025-06-18), OAuth 2.1, RFC 8252, RFC 7591, RFC 8707, RFC 9728, and observed behavior of real MCP clients tested against our Gateway.

---

## Requirement Classification

Every requirement in this document is tagged with one of the following:

| Tag | Meaning |
|-----|---------|
| **PROTOCOL** | Required by the MCP specification or referenced RFC. Non-negotiable. |
| **SECURITY** | Required for secure operation. Non-negotiable. |
| **DIPOLE** | Dipole Gateway design invariant. Our architectural choice, not mandated by protocol. |
| **COMPAT** | Needed because a major client breaks without it, even though the protocol doesn't strictly require it. |
| **QUIRK** | Observed client-specific behavior. Documented for awareness, no code change needed. |

---

## Architectural Principle: Zero-Branching Gateway

The MCP Identity Gateway MUST NOT contain client-specific branching logic. No `if client == "Claude"` or `if client == "Gemini"` anywhere in the codebase.

Apparent client incompatibilities stem from either:
1. **Gateway non-compliance** — returning responses that violate the MCP schema (e.g., `null` instead of `[]`)
2. **OAuth environment differences** — Web HTTPS callbacks vs. CLI ephemeral loopback ports vs. IDE custom schemes

By strictly adhering to the JSON-RPC schema and implementing RFC-compliant loopback normalization, a single generic codebase serves all clients.

The Gateway is unique in that it operates as both:
- **MCP Server** (left side) — serving AI clients (Claude, Gemini, Cursor)
- **MCP Client** (right side) — calling downstream MCP servers (Jira, Slack, custom servers)

Both sides must be protocol-compliant independently.

---

## 1. Tested MCP Clients

| Client | Type | Redirect URI Pattern | OAuth Flow | Tool Name Validation | Null Array Handling |
|--------|------|---------------------|-----------|---------------------|---------------------|
| Claude.ai | Web | `https://claude.ai/api/mcp/auth_callback` | Full DCR + PKCE | Rejects spaces in tool names | Accepts `null` (lenient) |
| Claude Desktop | Desktop | `http://localhost:<fixed-port>/callback` | Full DCR + PKCE | Same as Claude.ai | Same |
| Claude Code CLI | CLI | `http://localhost:<random-port>/callback` | DCR + PKCE | Same | Same |
| Gemini CLI | CLI | `http://localhost:<random-port>/oauth/callback` | DCR + PKCE + `/mcp auth` command | Accepts sanitized names | Rejects `null`, requires `[]` (strict) |
| Cursor | IDE | `http://localhost:8787/callback` (fixed) | DCR + PKCE | Unknown (untested) | Unknown |
| ChatGPT Web | Web | `https://chatgpt.com/aip/...` | DCR + PKCE | Unknown | Unknown |
| Windsurf | IDE | `http://localhost:<port>/callback` | DCR + PKCE | Unknown | Unknown |

---

## 2. Protocol Compliance Requirements

### 2.1 JSON-RPC Responses — Null vs Empty Array [PROTOCOL]

**Requirement:** All array fields in JSON-RPC responses MUST be empty arrays `[]`, never `null`.

**Rationale:** The MCP TypeScript schema defines `tools` and `resources` as arrays. Strict clients (Gemini CLI) validate against this schema and reject `null`. Lenient clients (Claude) accept both, but `null` is technically non-compliant.

**Affected responses:**
| Method | Field | Correct | Incorrect |
|--------|-------|---------|-----------|
| `tools/list` | `result.tools` | `[]` | `null` |
| `resources/list` | `result.resources` | `[]` | `null` |
| `resources/templates/list` | `result.resourceTemplates` | `[]` | `null` |
| `prompts/list` | `result.prompts` | `[]` | `null` |

**Implementation rule:** When aggregating from downstream servers and no items are collected, initialize the slice before returning:
```go
allTools := make([]MCPTool, 0) // NOT: var allTools []MCPTool
```

### 2.2 Tool Name Format [COMPAT]

**Requirement:** Exposed tool names MUST match the regex `^[a-zA-Z0-9_-]+$`. No spaces, no special characters, no unicode.

**Rationale:** Claude.ai explicitly rejects tool names containing spaces (shows "unsupported names" warning and excludes them). The MCP spec does not explicitly restrict characters, but all major clients enforce alphanumeric + underscore + hyphen. This is a compatibility requirement, not a protocol mandate.

**Our format:** `<sanitized_server_name>_<uuid_prefix>__<downstream_tool_name>`

Example: `customer_database_mcp_server_a36a9b13__list_customers`

**Sanitization rules:**
1. Lowercase the server name
2. Replace spaces with single underscore
3. Strip all characters not in `[a-z0-9_-]`
4. Collapse consecutive underscores to single underscore
5. Trim leading/trailing underscores
6. Append `_` + first 8 hex chars of ServerWorkload UUID (collision safety)
7. Separator between prefix and tool name: `__` (double underscore)
8. The sanitized prefix MUST NOT contain `__` (guaranteed by rule 4)

**Downstream tool names** are passed through unmodified — MCP servers are expected to produce spec-compliant names. We do NOT sanitize downstream tool names.

### 2.3 Redirect URI Handling (Loopback Clients) [PROTOCOL]

**Requirement:** The Authorization Server MUST ignore the port component when matching `redirect_uri` for loopback addresses (`localhost`, `127.0.0.1`, `[::1]`).

**Rationale:** Per RFC 8252 Section 7.3, native apps and CLI tools use ephemeral ports. The authorization server MUST allow any port for loopback redirect URIs. Gemini CLI uses a random port each invocation. Cursor uses a fixed port (8787). Claude.ai uses an HTTPS callback (not localhost).

**Spec reference:** OAuth 2.1 Draft + RFC 8252 Section 7.3:
> "The authorization server MUST allow any port to be specified at the time of the request for loopback IP redirect URIs."

**Implementation rule:** When comparing redirect URIs during authorization:
- If the registered URI host is `localhost`, `127.0.0.1`, or `[::1]`: match scheme + host + path only, ignore port
- If the registered URI uses HTTPS (non-loopback): exact match required (scheme + host + port + path)
- Treat `localhost`, `127.0.0.1`, and `[::1]` as equivalent loopback hosts

**Client-specific redirect URIs observed:**
| Client | Redirect URI | Port Behavior |
|--------|-------------|---------------|
| Claude.ai | `https://claude.ai/api/mcp/auth_callback` | Fixed (HTTPS, non-localhost) |
| Gemini CLI | `http://localhost:<random>/oauth/callback` | Random each session |
| Cursor | `http://localhost:8787/callback` | Fixed at 8787 |
| Claude Code | `http://localhost:<random>/callback` | Random |

### 2.4 Initialize Lifecycle [PROTOCOL]

**Requirement:** After sending `initialize` and receiving a successful response, the client MUST send a `notifications/initialized` notification before making any other requests.

**Rationale:** MCP spec (Lifecycle section): "The client MUST send an initialized notification to indicate it is ready to begin normal operations." Some servers reject subsequent requests without this notification.

**Our Gateway as server:** We don't enforce this (we accept requests without it). But our Gateway as a **client** to downstream servers MUST send it.

**Our Gateway as client to downstream servers:**
```
Gateway → downstream: initialize request
Gateway ← downstream: initialize response
Gateway → downstream: notifications/initialized notification
Gateway → downstream: tools/list request (now safe)
```

### 2.5 Session Management [PROTOCOL]

**Requirement:** If the server returns `Mcp-Session-Id` in the initialize response, the client MUST include it in all subsequent requests.

**Rationale:** Stateful MCP servers (like FastMCP-based ones) use session IDs to track client state. Requests without the session ID after initialization get rejected with 404.

**Our implementation:**
- As server (Gateway): We currently don't issue session IDs (stateless). Acceptable per spec — sessions are optional.
- As client (to downstream): We store and send back session IDs correctly.

### 2.6 OAuth Discovery Flow [PROTOCOL]

**Requirement:** The full OAuth discovery flow MUST be:
1. Client hits Gateway → 401 with `WWW-Authenticate: Bearer resource_metadata="<url>"`
2. Client fetches Protected Resource Metadata → gets `authorization_servers`
3. Client fetches AS Metadata → gets endpoints
4. Client registers via DCR (optional but SHOULD be supported)
5. Client authorizes via PKCE
6. Client exchanges code for token
7. Client uses Bearer token on all subsequent requests

**All clients tested follow this exact flow.** No client-specific deviations observed.

### 2.7 Token Audience (`aud` claim) [PROTOCOL]

**Requirement:** The access token's `aud` claim MUST be the canonical URI of the MCP server (the Gateway URL). Clients send this as the `resource` parameter in authorize/token requests per RFC 8707.

**Observed `resource` parameter values from clients:**
| Client | Resource Parameter |
|--------|-------------------|
| Claude.ai | `https://<tenant>-mcpgateway-staging.dipolehq.com/mcp` |
| Gemini CLI | `https://<tenant>-mcpgateway-staging.dipolehq.com/mcp` |
| curl (our test) | `https://<tenant>-mcpgateway-staging.dipolehq.com/mcp` |

**Our token `aud` claim:** `https://<tenant>-mcpgateway-staging.dipolehq.com/` (with trailing slash)

**Matching rule:** Trailing slash normalization MUST be applied when comparing `aud` against expected audience.

### 2.8 DCR (Dynamic Client Registration) [PROTOCOL]

**Requirement:** The Authorization Server SHOULD support RFC 7591 DCR. When `grant_types` and `response_types` are omitted from the registration request, defaults MUST be applied:
- `grant_types` default: `["authorization_code"]`
- `response_types` default: `["code"]`

**Rationale:** Claude.ai omits both fields. Gemini CLI sends them explicitly. Our DCR service must handle both cases.

### 2.9 PKCE [SECURITY]

**Requirement:** PKCE (RFC 7636) with S256 method MUST be required for all authorization code exchanges. Plain method MUST NOT be accepted.

**All tested clients use PKCE S256.** This is mandatory per MCP spec authorization section.

### 2.10 Token Refresh [PROTOCOL]

**Requirement:** If the Credential Provider has refresh enabled, the token response MUST include a `refresh_token`. Refresh tokens MUST be single-use (rotation). The authorization server metadata MUST advertise `"grant_types_supported": ["authorization_code", "refresh_token"]`.

---

## 3. Response Format Requirements

### 3.1 Successful tool call response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{"type": "text", "text": "..."}],
    "isError": false
  }
}
```

### 3.2 tools/list response (empty)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": []
  }
}
```

### 3.3 resources/list response (empty)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "resources": []
  }
}
```

### 3.4 Error responses

Protocol errors use standard JSON-RPC error codes:
| Code | Meaning |
|------|---------|
| -32700 | Parse error (malformed JSON) |
| -32600 | Invalid Request (wrong jsonrpc version, missing fields) |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

Tool execution errors use `isError: true` in the result, NOT JSON-RPC error codes:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{"type": "text", "text": "Error: permission denied"}],
    "isError": true
  }
}
```

---

## 4. Transport Requirements (Streamable HTTP)

### 4.1 Request headers the Gateway MUST accept from clients [PROTOCOL]

| Header | Required | Purpose |
|--------|----------|---------|
| `Content-Type: application/json` | Yes | JSON-RPC body |
| `Accept: application/json, text/event-stream` | Yes | Client must accept both |
| `Authorization: Bearer <token>` | Yes (after auth) | Access token |
| `Mcp-Session-Id: <id>` | Conditional | If server issued one |

### 4.2 Response content types the Gateway MUST support [PROTOCOL]

- `application/json` — single JSON-RPC response (simple case)
- `text/event-stream` — SSE stream with one or more events (required for streaming)

### 4.3 Status codes [PROTOCOL]

| Status | When |
|--------|------|
| 200 | Successful request (with JSON or SSE body) |
| 202 | Accepted — only for notifications/responses, NOT for requests with an `id` |
| 401 | No/invalid token (include `WWW-Authenticate` header) |
| 404 | Unknown tenant, or stale session ID |
| 400 | Malformed request |

---

## 5. Gateway Proxy Invariants [DIPOLE]

### 5.1 Transparent forwarding

The Gateway MUST NOT transform downstream response content. It forwards:
- `result` (including `structuredContent`, `_meta`, `annotations`)
- `content` arrays (text, image, audio, resource links)
- `isError` flags
- Tool `inputSchema` and `outputSchema`

The ONLY transformation is tool name prefixing (adding server prefix on list, stripping it on call).

### 5.2 Audit events use real names

Workload Event audit logs MUST contain the human-readable server name (`cred.ServerWorkload.Name`), never the internal sanitized routing prefix.

### 5.3 Error isolation

If one downstream server fails during `tools/list`, the Gateway silently skips it and returns tools from servers that succeeded.

### 5.4 Transient retry on initialize

Serverless/cold-start downstream servers may return empty responses on first request. Retry exactly once (2s delay) for transient errors only (network failures, empty SSE, 5xx). Never retry 4xx, JSON-RPC errors, or protocol violations.

---

## 6. Known Client Quirks [QUIRK]

### 6.1 Claude.ai
- Registers via DCR with `redirect_uris: ["https://claude.ai/api/mcp/auth_callback"]` — HTTPS, not localhost
- Omits `grant_types` and `response_types` in DCR request (we must apply defaults)
- Caches tools/list result — reconnect required to refresh
- Lenient on null arrays
- Uses display-friendly tool names in UI (strips prefix, replaces `_` with spaces)

### 6.2 Gemini CLI
- Uses random ephemeral port for localhost callback each session
- Path is `/oauth/callback` (not `/callback`)
- Requires explicit `/mcp auth <server>` command to trigger OAuth flow (doesn't auto-discover from 401 on first use)
- Strict JSON schema validation — rejects `null` where array expected
- Shows `"Error discovering resources"` if resources/list returns null
- Successfully loads tools even if resources discovery fails

### 6.3 Cursor
- Fixed port 8787 for localhost callback: `http://localhost:8787/callback`
- Also supports `cursor://anysphere.cursor-mcp/oauth/callback` (desktop-specific, not standard)
- Untested with our Gateway (planned)

### 6.4 ChatGPT Web
- Requires Business/Enterprise plan for custom MCP connectors
- Developer Mode must be enabled in settings
- Uses HTTPS callback URL (similar to Claude.ai pattern)
- Untested with our Gateway (plan-restricted)

---

## 7. Implementation Checklist

### Must fix now (blocking Gemini CLI):
- [ ] `handleResourcesList` returns `{"resources": []}` not `{"resources": null}`
- [ ] `handleToolsList` returns `{"tools": []}` not `{"tools": null}`
- [ ] Loopback redirect URI matching ignores port (RFC 8252 Section 7.3)

### Already implemented:
- [x] Tool names sanitized with UUID suffix (collision-safe)
- [x] `notifications/initialized` sent to downstream
- [x] Transient retry on initialize (cold-start)
- [x] DCR applies defaults for omitted fields
- [x] PKCE S256 required
- [x] Token `aud` trailing-slash normalization
- [x] 401 with `WWW-Authenticate` + `resource_metadata`
- [x] Session ID stored and forwarded to downstream
- [x] Audit events use human-readable server name
- [x] Transparent content forwarding (no transformation)

### Future (not MVP):
- [ ] Pagination support for tools/list with many tools
- [ ] `prompts/list` and `resources/templates/list` if downstream servers expose them
- [ ] Structured logging for downstream server failures
- [ ] Protocol version negotiation updates as MCP spec evolves
- [ ] Multi-version support (2025-06-18 / 2025-11-25 / 2026-07-28)

---

## 8. References

- [MCP Specification 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18)
- [MCP Authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
- [MCP Transports (Streamable HTTP)](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)
- [MCP Lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)
- [MCP Tools](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP Resources](https://modelcontextprotocol.io/specification/2025-06-18/server/resources)
- [OAuth 2.1 Draft (IETF)](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/)
- [RFC 8252 — OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252) (Section 7.3: loopback port)
- [RFC 7591 — Dynamic Client Registration](https://www.rfc-editor.org/rfc/rfc7591)
- [RFC 8707 — Resource Indicators for OAuth 2.0](https://www.rfc-editor.org/rfc/rfc8707)
- [RFC 9728 — OAuth 2.0 Protected Resource Metadata](https://www.rfc-editor.org/rfc/rfc9728)
- [RFC 8414 — OAuth 2.0 Authorization Server Metadata](https://www.rfc-editor.org/rfc/rfc8414)
- [Duende IdentityServer — Port-Agnostic Localhost Redirect URIs for MCP](https://duendesoftware.com/blog/20260707-port-agnostic-localhost-redirect-uris-mcp-auth)
