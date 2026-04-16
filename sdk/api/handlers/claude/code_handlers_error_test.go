package claude

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

type claudeCooldownTestExecutor struct{}

func (e *claudeCooldownTestExecutor) Identifier() string { return "claude" }

func (e *claudeCooldownTestExecutor) Execute(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, errors.New("Execute should not be called when auth is cooling down")
}

func (e *claudeCooldownTestExecutor) ExecuteStream(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	return nil, errors.New("ExecuteStream should not be called when auth is cooling down")
}

func (e *claudeCooldownTestExecutor) Refresh(ctx context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *claudeCooldownTestExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, errors.New("CountTokens should not be called when auth is cooling down")
}

func (e *claudeCooldownTestExecutor) HttpRequest(context.Context, *coreauth.Auth, *http.Request) (*http.Response, error) {
	return nil, errors.New("HttpRequest should not be called when auth is cooling down")
}

func newClaudeCooldownTestHandler(t *testing.T) *ClaudeCodeAPIHandler {
	t.Helper()

	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&claudeCooldownTestExecutor{})

	model := "claude-sonnet-4-6"
	next := time.Now().Add(45 * time.Second)
	auth := &coreauth.Auth{
		ID:       "claude-cooldown-auth",
		Provider: "claude",
		Status:   coreauth.StatusActive,
		ModelStates: map[string]*coreauth.ModelState{
			model: {
				Status:         coreauth.StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
				LastError: &coreauth.Error{
					HTTPStatus: http.StatusTooManyRequests,
					Message:    `{"error":{"message":"Resource has been exhausted (e.g. check quota).","status":"RESOURCE_EXHAUSTED"}}`,
				},
				Quota: coreauth.QuotaState{
					Exceeded:      true,
					Reason:        "quota",
					NextRecoverAt: next,
				},
			},
		},
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("Register auth: %v", err)
	}

	registry.GetGlobalRegistry().RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: model}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(auth.ID)
	})

	base := handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	return NewClaudeCodeAPIHandler(base)
}

func TestClaudeMessages_LocalCooldownReturnsAnthropicRateLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newClaudeCooldownTestHandler(t)

	router := gin.New()
	router.POST("/v1/messages", h.ClaudeMessages)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-6","max_tokens":64,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claude-Code-Session-Id", "session-1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
	if got := resp.Header().Get("Retry-After"); got == "" {
		t.Fatalf("Retry-After header = empty")
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body=%s", err, resp.Body.String())
	}
	if got, _ := payload["type"].(string); got != "error" {
		t.Fatalf("type = %q, want %q", got, "error")
	}
	rawErr, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing object: %v", payload)
	}
	if got, _ := rawErr["type"].(string); got != "rate_limit_error" {
		t.Fatalf("error.type = %q, want %q", got, "rate_limit_error")
	}
	if got, _ := rawErr["message"].(string); !strings.Contains(got, "Resource has been exhausted") {
		t.Fatalf("error.message = %q, want upstream quota message", got)
	}
	if strings.Contains(resp.Body.String(), "model_cooldown") {
		t.Fatalf("response body leaked internal model_cooldown payload: %s", resp.Body.String())
	}
}

func TestClaudeCountTokens_LocalCooldownReturnsAnthropicRateLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newClaudeCooldownTestHandler(t)

	router := gin.New()
	router.POST("/v1/messages/count_tokens", h.ClaudeCountTokens)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claude-Code-Session-Id", "session-2")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
	if got := resp.Header().Get("Retry-After"); got == "" {
		t.Fatalf("Retry-After header = empty")
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body=%s", err, resp.Body.String())
	}
	rawErr, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing object: %v", payload)
	}
	if got, _ := rawErr["type"].(string); got != "rate_limit_error" {
		t.Fatalf("error.type = %q, want %q", got, "rate_limit_error")
	}
}

func TestClaudeMessages_StreamBootstrapCooldownReturnsAnthropicRateLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newClaudeCooldownTestHandler(t)

	router := gin.New()
	router.POST("/v1/messages", h.ClaudeMessages)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-6","max_tokens":64,"stream":true,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claude-Code-Session-Id", "session-3")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
	if got := resp.Header().Get("Retry-After"); got == "" {
		t.Fatalf("Retry-After header = empty")
	}
	if strings.Contains(resp.Body.String(), "event: error") {
		t.Fatalf("bootstrap failure should return JSON error, got SSE body: %s", resp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body=%s", err, resp.Body.String())
	}
	rawErr, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing object: %v", payload)
	}
	if got, _ := rawErr["type"].(string); got != "rate_limit_error" {
		t.Fatalf("error.type = %q, want %q", got, "rate_limit_error")
	}
}

func TestForwardClaudeStream_TerminalRateLimitUsesAnthropicErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		t.Fatalf("gin test writer does not implement http.Flusher")
	}

	h := NewClaudeCodeAPIHandler(handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil))
	data := make(chan []byte, 1)
	errs := make(chan *interfaces.ErrorMessage, 1)
	data <- []byte("event: message_start\ndata: {}\n\n")
	close(data)
	errs <- &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error: &coreauth.Error{
			HTTPStatus: http.StatusTooManyRequests,
			Message:    `{"error":{"message":"Resource has been exhausted (e.g. check quota)."}}`,
		},
		Addon: http.Header{
			"Retry-After": {"30"},
		},
	}
	close(errs)

	h.forwardClaudeStream(c, flusher, func(error) {}, data, errs)

	body := recorder.Body.String()
	if !strings.Contains(body, "event: error") {
		t.Fatalf("body missing SSE error event: %s", body)
	}
	if !strings.Contains(body, `"type":"rate_limit_error"`) {
		t.Fatalf("body missing Claude rate_limit_error: %s", body)
	}
	if !strings.Contains(body, "Retry after 30 seconds") {
		t.Fatalf("body missing retry hint: %s", body)
	}
}
