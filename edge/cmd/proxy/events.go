/* Workload event emission for the agent proxy. */
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/protocols"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

func emitWorkloadEvent(cloud *cloudclient.CloudClient, proto protocols.Protocol, targetHost string, targetPort int, sourceIP string, sourcePort int, proxyPort int, connID uuid.UUID, credInjected bool, httpInfo *protocols.HTTPRequestInfo, resourceSetID string, policyID string, clientWorkloadID string, serverWorkloadID string, logger *slog.Logger) {
	eventType := protocolToEventType(proto)

	outcome := "Passthrough"
	if credInjected {
		outcome = "Modified"
	}
	if httpInfo != nil && httpInfo.Injected {
		outcome = "Modified"
	}

	data := &protocols.EventData{
		EventType:        eventType,
		ConnectionID:     connID.String(),
		PolicyID:         policyID,
		ClientWorkloadID: clientWorkloadID,
		ServerWorkloadID: serverWorkloadID,
		ResourceSetID:    resourceSetID,
		Outcome:          outcome,
		TargetHost:       targetHost,
		TargetPort:       targetPort,
		SourceIP:         sourceIP,
		SourcePort:       sourcePort,
		ProxyPort:        proxyPort,
	}

	if httpInfo != nil {
		data.HTTPMethod = httpInfo.Method
		data.HTTPUri = httpInfo.URI
		data.HTTPStatus = httpInfo.StatusCode
		data.Headers = httpInfo.Headers
	}

	event := protocols.BuildWorkloadEvent(data)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cloud.IngestEvents(ctx, []*pb.WorkloadEvent{event}); err != nil {
		logger.Debug("workload event emission failed", "error", err, "connection_id", connID)
	}
}

func protocolToEventType(proto protocols.Protocol) string {
	switch proto {
	case protocols.ProtoHTTPS, protocols.ProtoHTTP, protocols.ProtoMCP:
		return "http.request"
	case protocols.ProtoPostgreSQL:
		return "postgres.request"
	case protocols.ProtoMySQL:
		return "mysql.request"
	case protocols.ProtoRedis:
		return "redis.request"
	case protocols.ProtoSnowflake:
		return "snowflake.request"
	case protocols.ProtoRedshift:
		return "redshift.request"
	case protocols.ProtoOracle:
		return "oracleDatabase.request"
	default:
		return "tcp.open"
	}
}
