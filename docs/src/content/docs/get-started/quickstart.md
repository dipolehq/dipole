---
title: Quickstart
description: Get Dipole running in under 15 minutes.
sidebar:
  order: 2
---

This guide walks you through deploying Dipole on Kubernetes and creating your first access policy.

## Prerequisites

- A Kubernetes cluster (1.28+)
- Helm 3.x installed
- `kubectl` configured for your cluster

## Install Dipole

```bash
helm repo add dipole https://helm.dipolehq.com
helm install dipole dipole/dipole \
  --set tenant=<your-tenant-id> \
  --set agentController.deviceCode=<your-device-code>
```

## Create Your First Policy

1. Log in to the Dipole admin console
2. Create a Client Workload (e.g., your application pod)
3. Create a Server Workload (e.g., `api.stripe.com:443`)
4. Create a Credential Provider (e.g., API Key with your Stripe key)
5. Create a Trust Provider (e.g., Kubernetes Service Account)
6. Create an Access Policy linking them together
7. Annotate your pod with `dipolehq.com/agent-inject: "enabled"`

Your application will now have credentials injected transparently on every request to Stripe.
