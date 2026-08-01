---
title: Kubernetes Annotations
description: Annotations for configuring Dipole sidecar injection.
sidebar:
  order: 2
---

## Pod Annotations

| Annotation | Required | Description |
|-----------|----------|-------------|
| `dipolehq.com/agent-inject: "enabled"` | Yes | Triggers Agent Proxy sidecar injection |
| `dipolehq.com/resource-set-id` | No | Scope to specific Resource Set |
| `dipolehq.com/client-id` | No | Set Client Workload ID |
| `dipolehq.com/steering-mode` | No | `transparent` (default) or `explicit` |
| `dipolehq.com/agent-proxy-env-<VAR>` | No | Set env var on proxy container |

## Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    metadata:
      annotations:
        dipolehq.com/agent-inject: "enabled"
        dipolehq.com/steering-mode: "transparent"
    spec:
      containers:
        - name: myapp
          image: myapp:latest
```

When the pod is created, the Agent Injector webhook automatically adds:

1. An init container that sets up iptables rules for traffic interception
2. The Agent Proxy sidecar container
3. A shared volume for the tenant CA certificate
