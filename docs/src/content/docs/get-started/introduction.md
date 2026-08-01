---
title: Introduction
description: What Dipole is and the problem it solves.
sidebar:
  order: 1
---

Dipole is a Workload Identity and Access Management platform for Non-Human Identities. It eliminates static secrets from workload-to-workload communication by replacing them with identity-based, policy-driven, just-in-time credential injection.

## How It Works

1. A client workload makes an outbound request
2. The Agent Proxy intercepts the request transparently
3. The proxy gathers identity evidence from the runtime environment
4. Dipole Cloud evaluates the access policy and verifies identity
5. On success, a credential is retrieved and injected into the request
6. The client workload never sees, stores, or manages the credential

## Key Concepts

- **Client Workloads** — services that initiate access requests
- **Server Workloads** — APIs, databases, or SaaS services being accessed
- **Trust Providers** — verify workload identity via cryptographic attestation
- **Credential Providers** — retrieve or generate credentials just-in-time
- **Access Policies** — define who can access what, under what conditions
- **Access Conditions** — contextual checks (time, GeoIP) evaluated at request time
