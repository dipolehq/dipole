/*
redshift.go handles Amazon Redshift connections on port 5439. Redshift uses
the PostgreSQL wire protocol with IAM-based authentication. Reuses the
PostgreSQL handler — the credential (username + temporary token) comes from
the AWS STS Credential Provider.
*/
package protocols

import (
	"log/slog"
	"net"

	"github.com/google/uuid"
)

type RedshiftHandler struct {
	pg *PostgresHandler
}

func NewRedshiftHandler(logger *slog.Logger) *RedshiftHandler {
	return &RedshiftHandler{
		pg: &PostgresHandler{Logger: logger},
	}
}

func (h *RedshiftHandler) Handle(clientConn, serverConn net.Conn, cred *PGCredential, connID uuid.UUID) {
	h.pg.Handle(clientConn, serverConn, cred, connID)
}
