---
title: MCP Overview
description: Secure AI agent access to tools and services.
sidebar:
  order: 1
---

Dipole provides identity and access management for AI agents communicating via the Model Context Protocol (MCP).

## The Problem

AI agents (Claude, Gemini, etc.) connect to MCP servers that expose tools (send email, query DB, create ticket). These MCP servers need authentication, but:

- Storing long-lived credentials in MCP server configs is insecure
- No way to distinguish which user instructed the agent
- No per-user credential isolation

## Dipole's Solution

### MCP Authorization Server

An OAuth 2.1 Authorization Server that enables MCP clients to authenticate and get scoped tokens. Supports:

- Dynamic Client Registration (DCR)
- PKCE (required)
- IdP-based user authentication
- Blended identity (workload + user in a single policy)

### MCP Identity Gateway

A managed proxy between AI agents and MCP servers that:

- Enforces access policies per request
- Injects per-user credentials (each user gets their own MCP server tokens)
- Provides full audit trail of which user, via which agent, accessed which tool
- Ensures AI agents never hold direct credentials to MCP servers
