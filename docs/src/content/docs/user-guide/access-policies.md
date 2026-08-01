---
title: Access Policies
description: Define who can access what, under what conditions.
sidebar:
  order: 1
---

An Access Policy is the fundamental unit of authorization in Dipole. It links a Client Workload to a Server Workload, specifying how identity is verified and what credential is injected.

## Policy Components

- **Client Workload** — who is making the request
- **Server Workload** — what is being accessed
- **Trust Providers** — how identity is verified (AND logic — all must pass)
- **Credential Provider** — what credential to inject
- **Access Conditions** — additional contextual checks (AND logic — all must pass)

## Policy Evaluation Flow

1. Agent Proxy intercepts a request and sends identity evidence to Cloud
2. Cloud matches the request against active policies
3. Trust Provider attestation is verified
4. Access Conditions are evaluated
5. Credential is retrieved and returned to the proxy
6. If any step fails, the request is denied (fail-closed)

## Creating a Policy

Navigate to **Access Policies** in the admin console and click **Create Policy**. Use the multi-tab Policy Builder to configure each component.
