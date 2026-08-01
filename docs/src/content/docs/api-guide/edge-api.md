---
title: Edge API
description: Programmatic credential retrieval for CI/CD and serverless.
sidebar:
  order: 2
---

The Edge API enables workloads to retrieve credentials without an Agent Proxy. Used primarily by CI/CD pipelines and serverless functions.

## Base URL

```
https://<tenant-id>.ec.<region>.dipolehq.com
```

## POST /edge/v1/auth

Authenticate a workload using attestation evidence. No `Authorization` header — identity is proved via the request body.

```json
{
  "clientId": "dipole:ap-south-1:acme:identity:github_idtoken:uuid",
  "client": {
    "github": {
      "identityToken": "eyJhbG..."
    }
  }
}
```

**Response:**

```json
{
  "accessToken": "eyJhbG...",
  "tokenType": "Bearer",
  "expiresIn": 3600
}
```

## POST /edge/v1/credentials

Retrieve a credential for a specific Server Workload. Requires a Bearer token from `/edge/v1/auth`.

```json
{
  "server": {
    "host": "api.stripe.com",
    "port": 443
  }
}
```

**Response:**

```json
{
  "credentialType": "ApiKey",
  "expiresAt": "2026-01-15T10:30:00Z",
  "data": {
    "apiKey": "sk_live_abc123..."
  }
}
```
