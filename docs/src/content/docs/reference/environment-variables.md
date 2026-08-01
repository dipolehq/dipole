---
title: Environment Variables
description: All configurable environment variables for Dipole Edge components.
sidebar:
  order: 1
---

## Agent Proxy

| Variable | Default | Description |
|----------|---------|-------------|
| `DIPOLE_AGENT_CONTROLLER` | — | Agent Controller URL |
| `DIPOLE_HTTP_SERVER_PORT` | `8000` | HTTP proxy port |
| `DIPOLE_LOG_LEVEL` | `info` | Log level (error, warn, info, debug, trace) |
| `DIPOLE_METRICS_PORT` | `9099` | Prometheus metrics port |
| `DIPOLE_METRICS_ENABLED` | `true` | Enable/disable metrics |
| `DIPOLE_PASS_THROUGH_TRAFFIC_BEFORE_REGISTRATION` | `true` | Allow traffic before Cloud registration |
| `DIPOLE_SIGTERM_STRATEGY` | `immediate` | Shutdown strategy |
| `DIPOLE_STEERING_ALLOWED_HOSTS` | — | Comma-separated hosts for selective steering |
| `DIPOLE_HTTP_IDLE_TIMEOUT_SECS` | `3600` | HTTP idle connection timeout |
| `DIPOLE_RESOURCE_SET_ID` | — | Associate proxy with Resource Set |
| `CLIENT_WORKLOAD_ID` | — | Associate proxy with Client Workload |
| `K8S_POD_NAME` | — | Injected via Kubernetes Downward API |
| `K8S_NAMESPACE` | — | Injected via Kubernetes Downward API |

## Agent Controller

| Variable | Default | Description |
|----------|---------|-------------|
| `DIPOLE_TENANT_ID` | — | Tenant ID for registration |
| `DIPOLE_DEVICE_CODE` | — | Device code for registration |
| `DIPOLE_AGENT_CONTROLLER_ID` | — | Controller ID for Trust Provider registration |
| `DIPOLE_LOG_LEVEL` | `information` | Log level |
| `DIPOLE_STACK_DOMAIN` | `dipolehq.com` | Cloud domain |
