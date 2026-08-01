---
title: MCP Identity Gateway
description: Zero-credential-exposure proxy for AI agent MCP traffic.
sidebar:
  order: 2
---

The MCP Identity Gateway sits between AI agents and MCP servers, enforcing policies and injecting per-user credentials.

## Two-Policy Model

The Gateway requires two Access Policies:

### Policy 1: Client-to-Gateway

- **Client Workload** — AI agent identified by redirect URI
- **Server Workload** — The Gateway endpoint
- **Trust Provider** — OIDC token validating the human user
- **Credential Provider** — OIDC token embedding user identity

### Policy 2: Gateway-to-Server

- **Client Workload** — The Gateway itself
- **Server Workload** — The actual MCP server
- **Trust Provider** — Validates the Gateway's token
- **Credential Provider** — Per-user OAuth token for the MCP server

## Per-User Credential Isolation

Each user gets their own MCP server credentials:

1. User authenticates via IdP when connecting through the Gateway
2. First-time access triggers OAuth consent with the MCP server
3. Dipole stores each user's token separately
4. Subsequent requests inject that user's specific token
5. Audit trail shows exactly which user accessed which tool
