---
title: Credential Providers
description: Retrieve or generate credentials just-in-time.
sidebar:
  order: 3
---

Credential Providers are the systems that produce the credential injected into outbound requests. They answer: "What credential does the Server Workload expect?"

## Credential Types

### Stored Credentials
- **API Key** — static API key stored encrypted in Dipole
- **Username & Password** — stored encrypted in Dipole

### Token Generation
- **OAuth 2.0 Client Credentials** — exchanges client credentials for an access token
- **OIDC ID Token** — generates an OIDC-compliant JWT signed by Dipole
- **JSON Web Token** — generates a custom JWT with configurable claims

### Cloud Federation
- **AWS STS Federation** — temporary AWS credentials via OIDC-based WIF
- **Azure Entra WIF** — Azure tokens via OIDC federation
- **Google GCP WIF** — GCP tokens via OIDC federation

### External Stores
- **HashiCorp Vault** — retrieves a Vault client token
- **AWS Secrets Manager** — retrieves secret values from AWS

## Credential Lifecycle

Credentials are short-lived and scoped. They are retrieved just-in-time, injected by the proxy, and never stored in the client workload's environment.
