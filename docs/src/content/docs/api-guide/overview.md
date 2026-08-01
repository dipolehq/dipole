---
title: API Overview
description: Dipole Cloud REST API for programmatic resource management.
sidebar:
  order: 1
---

The Dipole Cloud API provides full CRUD access to all platform resources.

## Base URL

```
https://<tenant-id>.dipolehq.com/api/v1
```

## Authentication

Include a Bearer token in the `Authorization` header:

```
Authorization: Bearer <api-token>
```

Generate API tokens from the Profile page in the admin console.

## Resource Set Scoping

Optionally scope requests to a specific Resource Set:

```
X-Dipole-ResourceSet: <resource-set-uuid>
```

## Available Resources

| Endpoint | Description |
|----------|-------------|
| `/api/v1/client-workloads` | Manage client workloads |
| `/api/v1/server-workloads` | Manage server workloads |
| `/api/v1/trust-providers` | Manage trust providers |
| `/api/v1/credential-providers` | Manage credential providers |
| `/api/v1/access-policies` | Manage access policies |
| `/api/v1/access-conditions` | Manage access conditions |
| `/api/v1/agent-controllers` | Manage agent controllers |
| `/api/v1/users` | Manage users |
| `/api/v1/roles` | Manage roles |
| `/api/v1/resource-sets` | Manage resource sets |

## Response Format

All responses use `camelCase` JSON field names.
