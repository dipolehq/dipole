---
title: Installation
description: Install the Dipole CLI.
sidebar:
  order: 1
---

The Dipole CLI enables programmatic credential retrieval for CI/CD pipelines and serverless environments where the Agent Proxy is not deployed.

## Download

```bash
# Linux amd64
curl -Lo dipole https://releases.dipolehq.com/cli/latest/dipole-linux-amd64
chmod +x dipole
sudo mv dipole /usr/local/bin/

# macOS arm64
curl -Lo dipole https://releases.dipolehq.com/cli/latest/dipole-darwin-arm64
chmod +x dipole
sudo mv dipole /usr/local/bin/
```

## Verify Installation

```bash
dipole --version
```

## Authentication

The CLI authenticates using OIDC tokens from your CI/CD platform (GitHub Actions, GitLab CI, etc.) rather than stored credentials.
