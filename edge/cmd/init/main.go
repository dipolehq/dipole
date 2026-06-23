/*
Init container binary that sets up iptables NAT REDIRECT rules for transparent
traffic interception. Runs as UID 26248 with NET_ADMIN capability, configures
the DIPOLE_REDIRECT chain, and exits. If any rule fails, exits non-zero to
prevent the pod from starting (fail-closed).
*/
package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/coreos/go-iptables/iptables"
)

const (
	table    = "nat"
	chain    = "DIPOLE_REDIRECT"
	outChain = "OUTPUT"
	toPort   = "15001"
	proxyUID = "65534"
)

var redirectPorts = []string{"443", "80", "8443", "5432", "3306", "6379", "1521"}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	proxyPort := envStr("DIPOLE_PROXY_PORT", toPort)
	proxyUid := envStr("DIPOLE_PROXY_UID", proxyUID)
	allowedHosts := os.Getenv("DIPOLE_STEERING_ALLOWED_HOSTS")

	ipt, err := iptables.New()
	if err != nil {
		logger.Error("failed to initialize iptables", "error", err)
		os.Exit(1)
	}

	if err := setupRules(ipt, proxyPort, proxyUid, allowedHosts, logger); err != nil {
		logger.Error("failed to set up iptables rules", "error", err)
		os.Exit(1)
	}

	logger.Info("iptables rules configured successfully", "proxy_port", proxyPort, "proxy_uid", proxyUid)
}

func setupRules(ipt *iptables.IPTables, proxyPort, proxyUid, allowedHosts string, logger *slog.Logger) error {
	if err := ipt.NewChain(table, chain); err != nil {
		// Chain may already exist — ignore
		logger.Debug("chain creation (may already exist)", "error", err)
	}

	// Exclude proxy's own traffic to prevent redirect loops
	if err := ipt.AppendUnique(table, chain, "-m", "owner", "--uid-owner", proxyUid, "-j", "RETURN"); err != nil {
		return fmt.Errorf("add UID exclusion rule: %w", err)
	}

	if allowedHosts != "" {
		// Selective steering: only redirect traffic to specific IPs
		if err := addSelectiveRules(ipt, proxyPort, allowedHosts, logger); err != nil {
			return err
		}
	} else {
		// Full steering: redirect all traffic on common ports
		for _, port := range redirectPorts {
			if err := ipt.AppendUnique(table, chain, "-p", "tcp", "--dport", port, "-j", "REDIRECT", "--to-ports", proxyPort); err != nil {
				return fmt.Errorf("add redirect for port %s: %w", port, err)
			}
		}
	}

	// Attach chain to OUTPUT
	if err := ipt.AppendUnique(table, outChain, "-j", chain); err != nil {
		return fmt.Errorf("attach chain to OUTPUT: %w", err)
	}

	return nil
}

func addSelectiveRules(ipt *iptables.IPTables, proxyPort, allowedHosts string, logger *slog.Logger) error {
	hosts := strings.Split(allowedHosts, ",")
	for _, h := range hosts {
		host := strings.TrimSpace(h)
		if host == "" {
			continue
		}

		ips, err := net.LookupHost(host)
		if err != nil {
			logger.Warn("failed to resolve host for selective steering", "host", host, "error", err)
			continue
		}

		for _, ip := range ips {
			if err := ipt.AppendUnique(table, chain, "-p", "tcp", "-d", ip, "-j", "REDIRECT", "--to-ports", proxyPort); err != nil {
				return fmt.Errorf("add selective redirect for %s (%s): %w", host, ip, err)
			}
			logger.Info("selective steering rule added", "host", host, "ip", ip)
		}
	}
	return nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
