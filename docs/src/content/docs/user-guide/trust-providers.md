---
title: Trust Providers
description: Verify workload identity through cryptographic attestation.
sidebar:
  order: 2
---

Trust Providers answer the question: "Can the runtime environment prove this workload's identity?"

Instead of shared secrets, the workload's environment provides cryptographic proof of identity.

## Supported Types

| Type | Environment | What It Proves |
|------|-------------|----------------|
| Kubernetes Service Account | K8s pods | Pod is running as a specific service account |
| AWS Metadata Service | EC2 instances | Workload runs on a specific EC2 instance |
| AWS Role | AWS workloads | Workload has a specific IAM role |
| GitHub Actions | GitHub CI/CD | Code runs in a specific repository/workflow |
| GitLab CI/CD | GitLab CI/CD | Code runs in a specific project/pipeline |
| OIDC ID Token | Any OIDC provider | Identity provider vouches for this workload |

## Match Rules

Each Trust Provider can have match rules that constrain which workloads pass attestation. All match rules use AND logic — every rule must pass.

Example: A Kubernetes Trust Provider with rules `namespace = production` AND `serviceaccount = myapp-sa` will only match pods with that exact namespace and service account.
