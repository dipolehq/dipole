---
title: Architecture
description: High-level overview of Dipole's architecture.
sidebar:
  order: 3
---

Dipole consists of two primary systems: the Cloud (control plane) and the Edge (data plane).

## Dipole Cloud

The SaaS control plane that:

- Evaluates all access policies
- Validates workload identities via Trust Providers
- Retrieves credentials from external systems
- Stores configuration and logs events
- Serves the admin UI and management APIs

## Dipole Edge

Components deployed in your environment:

- **Agent Proxy** — sidecar that intercepts traffic and injects credentials
- **Agent Controller** — manages proxy registration with Cloud
- **Agent Injector** — Kubernetes webhook that injects the proxy as a sidecar

## Communication

- All Edge-to-Cloud communication is outbound only (port 443)
- gRPC/HTTP2 for persistent policy evaluation stream
- No inbound ports required on any Edge component
