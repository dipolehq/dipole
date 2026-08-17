package mcp_gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func readJSONRPC(t *testing.T, contentType string, body string, requestID int64) ([]byte, error) {
	t.Helper()
	return readJSONRPCResponse(strings.NewReader(body), contentType, requestID)
}

func TestReadJSONRPCResponse_PlainJSON(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
	got, err := readJSONRPC(t, "application/json", body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != body {
		t.Errorf("expected body unchanged, got %s", got)
	}
}

func TestReadJSONRPCResponse_SSE_SingleEvent(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[]}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_NoEventLine(t *testing.T) {
	body := "data: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":2,"result":{}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_NotificationBeforeResponse(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\n" +
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"tools\":[{\"name\":\"foo\"}]}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"foo"}]}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_ResponseBeforeTrailingNotification(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"tools\":[{\"name\":\"foo\"}]}}\n\n" +
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"foo"}]}}`
	if string(got) != expected {
		t.Errorf("expected the id-matching event regardless of position, got %s", got)
	}
}

func TestReadJSONRPCResponse_SSE_MultipleResponses_PicksMatchingID(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"marker\":\"wrong\"}}\n\n" +
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"marker\":\"right\"}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed struct {
		Result struct {
			Marker string `json:"marker"`
		} `json:"result"`
	}
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Result.Marker != "right" {
		t.Errorf("expected the event matching id=2, got marker=%s", parsed.Result.Marker)
	}
}

func TestReadJSONRPCResponse_SSE_StringID_DoesNotMatchNumeric(t *testing.T) {
	body := "data: {\"jsonrpc\":\"2.0\",\"id\":\"7\",\"result\":{}}\n\n"
	_, err := readJSONRPC(t, "text/event-stream", body, 7)
	if err == nil {
		t.Fatal("expected no match between numeric request id 7 and string id \"7\", got a match")
	}
}

func TestReadJSONRPCResponse_SSE_NoMatchingID_Errors(t *testing.T) {
	body := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\n" +
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n"
	_, err := readJSONRPC(t, "text/event-stream", body, 99)
	if err == nil {
		t.Fatal("expected an error when no SSE event matches the request id, got nil")
	}
}

func TestReadJSONRPCResponse_SSE_EmptyBody_Errors(t *testing.T) {
	_, err := readJSONRPC(t, "text/event-stream", "", 1)
	if err == nil {
		t.Fatal("expected an error for an SSE response with no data: events, got nil")
	}
}

func TestReadJSONRPCResponse_SSE_MultiLineData(t *testing.T) {
	body := "data: {\"jsonrpc\":\"2.0\",\ndata: \"id\":1,\"result\":{}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "{\"jsonrpc\":\"2.0\",\n\"id\":1,\"result\":{}}"
	if string(got) != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_NoTrailingNewline(t *testing.T) {
	body := "data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}"
	got, err := readJSONRPC(t, "text/event-stream", body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":1,"result":{}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_WithCharset(t *testing.T) {
	body := "data: {\"jsonrpc\":\"2.0\",\"id\":5,\"result\":{\"ok\":true}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream; charset=utf-8", body, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":5,"result":{"ok":true}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_CommentsAndRetryIgnored(t *testing.T) {
	body := ": this is a comment\nretry: 3000\nevent: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n"
	got, err := readJSONRPC(t, "text/event-stream", body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":1,"result":{}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestReadJSONRPCResponse_SSE_ReturnsAsSoonAsMatchFound(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	go func() {
		fmt.Fprint(pw, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":3,\"result\":{\"ok\":true}}\n\n")
	}()

	done := make(chan struct{})
	var got []byte
	var err error
	go func() {
		got, err = readJSONRPCResponse(pr, "text/event-stream", 3)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readJSONRPCResponse did not return promptly after the matching event arrived; it appears to be waiting for EOF")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":3,"result":{"ok":true}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

type sseServer struct {
	mu             sync.Mutex
	sessionID      string
	handlers       map[string]func(id int64) (any, error)
	invalidateOnce atomic.Bool
}

func newSSEServer() *sseServer {
	return &sseServer{
		sessionID: "test-session-abc",
		handlers:  map[string]func(id int64) (any, error){},
	}
}

func (s *sseServer) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if req.Method != "initialize" {
			sid := r.Header.Get("Mcp-Session-Id")
			if sid == "" {
				http.Error(w, "missing session id", http.StatusBadRequest)
				return
			}
			if s.invalidateOnce.CompareAndSwap(true, false) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			s.mu.Lock()
			valid := sid == s.sessionID
			s.mu.Unlock()
			if !valid {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}

		var result any
		var err error
		if req.Method == "initialize" {
			result = map[string]any{"protocolVersion": "2025-03-26"}
		} else if h, ok := s.handlers[req.Method]; ok {
			result, err = h(req.ID)
		} else {
			err = fmt.Errorf("method not found")
		}

		w.Header().Set("Content-Type", "application/json")
		if req.Method == "initialize" {
			s.mu.Lock()
			sid := s.sessionID
			s.mu.Unlock()
			w.Header().Set("Mcp-Session-Id", sid)
		}

		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if err != nil {
			resp["error"] = map[string]any{"code": -32601, "message": err.Error()}
		} else {
			resp["result"] = result
		}
		respBody, _ := json.Marshal(resp)
		w.Write(respBody)
	}
}

func newTestClient(t *testing.T, srv *httptest.Server) *HTTPMCPClient {
	t.Helper()
	return &HTTPMCPClient{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
}

func TestHTTPMCPClient_ListTools_HappyPath(t *testing.T) {
	s := newSSEServer()
	s.handlers["tools/list"] = func(id int64) (any, error) {
		return map[string]any{"tools": []MCPTool{{Name: "foo"}}}, nil
	}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()

	client := newTestClient(t, srv)
	tools, _, err := client.ListTools(context.Background(), "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "foo" {
		t.Errorf("unexpected tools: %+v", tools)
	}
}

func TestHTTPMCPClient_ConcurrentCalls_InitializeOnlyOnce(t *testing.T) {
	s := newSSEServer()
	var initCount atomic.Int64
	s.handlers["tools/list"] = func(id int64) (any, error) {
		return map[string]any{"tools": []MCPTool{}}, nil
	}

	origHandler := s.handler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var peek struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &peek)
		if peek.Method == "initialize" {
			initCount.Add(1)
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		origHandler(w, r)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := client.ListTools(context.Background(), "tok")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("unexpected error from concurrent call: %v", err)
		}
	}

	if got := initCount.Load(); got != 1 {
		t.Errorf("expected exactly 1 initialize call across %d concurrent callers, got %d", n, got)
	}
}

func TestHTTPMCPClient_StaleSession_RetriesAndSucceeds(t *testing.T) {
	s := newSSEServer()
	s.handlers["tools/list"] = func(id int64) (any, error) {
		return map[string]any{"tools": []MCPTool{{Name: "recovered"}}}, nil
	}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()

	client := newTestClient(t, srv)

	if _, _, err := client.ListTools(context.Background(), "tok"); err != nil {
		t.Fatalf("initial call failed: %v", err)
	}

	s.invalidateOnce.Store(true)

	tools, _, err := client.ListTools(context.Background(), "tok")
	if err != nil {
		t.Fatalf("expected the client to recover from a stale session automatically, got error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "recovered" {
		t.Errorf("expected the retried call's real result, got: %+v", tools)
	}
}

func TestHTTPMCPClient_404WithoutSession_IsNotTreatedAsSessionExpiry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, _, err := client.ListTools(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected an error for a 404 with no session involved, got nil")
	}
}

func TestHTTPMCPClient_202OnRequest_IsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var peek struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &peek)
		if peek.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sid")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, _, err := client.CallTool(context.Background(), "tok", "some_tool", nil)
	if err == nil {
		t.Fatal("expected 202 Accepted on a tools/call request to be an error, got nil")
	}
}

func TestHTTPMCPClient_ContextDeadline_BoundsSlowServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var peek struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &peek)
		if peek.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sid")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
			return
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := client.ListTools(ctx, "tok")
	if err == nil {
		t.Fatal("expected the call to fail once the caller's context deadline elapsed, got nil")
	}
}
