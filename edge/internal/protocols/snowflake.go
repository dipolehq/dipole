/*
snowflake.go implements Snowflake credential injection via HTTP body
modification. Detects Snowflake by hostname (*.snowflakecomputing.com),
intercepts POST requests to /session/v1/login-request, and injects
LOGIN_NAME, TOKEN, and AUTHENTICATOR fields into the JSON body.
Non-login requests pass through unchanged.
*/
package protocols

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type SnowflakeHandler struct {
	Logger *slog.Logger
}

type SnowflakeCredential struct {
	Username string
	Token    string
}

func (h *SnowflakeHandler) Handle(clientReader *bufio.Reader, clientWriter io.Writer, serverConn net.Conn, cred *SnowflakeCredential, connID uuid.UUID) {
	serverReader := bufio.NewReader(serverConn)

	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			return
		}

		if req.Method == "POST" && req.URL.Path == "/session/v1/login-request" {
			h.injectLoginCredentials(req, cred)
		}

		if err := req.Write(serverConn); err != nil {
			h.Logger.Error("snowflake: forward request", "error", err, "connection_id", connID)
			return
		}

		resp, err := http.ReadResponse(serverReader, req)
		if err != nil {
			h.Logger.Error("snowflake: read response", "error", err, "connection_id", connID)
			return
		}

		if err := resp.Write(clientWriter); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()
	}
}

func (h *SnowflakeHandler) injectLoginCredentials(req *http.Request, cred *SnowflakeCredential) {
	if req.Body == nil {
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		// Not valid JSON — send original
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return
	}

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		data = make(map[string]interface{})
	}

	data["LOGIN_NAME"] = cred.Username
	data["TOKEN"] = cred.Token
	data["AUTHENTICATOR"] = "SNOWFLAKE_JWT"
	body["data"] = data

	modified, err := json.Marshal(body)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return
	}

	req.Body = io.NopCloser(bytes.NewReader(modified))
	req.ContentLength = int64(len(modified))
	req.Header.Set("Content-Length", strconv.Itoa(len(modified)))
}
