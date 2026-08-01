---
title: Retrieve Credentials
description: Use the CLI to retrieve credentials for CI/CD pipelines.
sidebar:
  order: 2
---

The `dipole credentials get` command retrieves a credential from Dipole Cloud using your platform's OIDC token for attestation.

## Usage

```bash
dipole credentials get \
  --client-id dipole:ap-south-1:acme:identity:github_idtoken:uuid \
  --server-workload-host api.stripe.com \
  --server-workload-port 443 \
  --id-token $GITHUB_OIDC_TOKEN
```

## Output Format

By default, the CLI outputs shell export format:

```bash
export TOKEN='sk_live_abc123...'
```

Use with `eval`:

```bash
eval $(dipole credentials get --client-id ... --id-token ...)
```

## Options

| Flag | Description |
|------|-------------|
| `--client-id` | Edge SDK Client ID (identifies tenant + Trust Provider) |
| `--server-workload-host` | Target service hostname |
| `--server-workload-port` | Target service port |
| `--id-token` | OIDC token from platform |
| `--output-format` | `sh-export`, `sh-env`, `powershell-env` |
| `--resource-set-id` | Scope to specific Resource Set |
