// Package claude provides HTTP handlers for Claude API code-related functionality.
// This package implements Claude-compatible streaming chat completions with sophisticated
// client rotation and quota management systems to ensure high availability and optimal
// resource utilization across multiple backend clients. It handles request translation
// between Claude API format and the underlying Gemini backend, providing seamless
// API compatibility while maintaining robust error handling and connection management.
package claude

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// ClaudeCodeAPIHandler contains the handlers for Claude API endpoints.
// It holds a pool of clients to interact with the backend service.
type ClaudeCodeAPIHandler struct {
	*handlers.BaseAPIHandler
}

// NewClaudeCodeAPIHandler creates a new Claude API handlers instance.
// It takes an BaseAPIHandler instance as input and returns a ClaudeCodeAPIHandler.
//
// Parameters:
//   - apiHandlers: The base API handler instance.
//
// Returns:
//   - *ClaudeCodeAPIHandler: A new Claude code API handler instance.
func NewClaudeCodeAPIHandler(apiHandlers *handlers.BaseAPIHandler) *ClaudeCodeAPIHandler {
	return &ClaudeCodeAPIHandler{
		BaseAPIHandler: apiHandlers,
	}
}

// HandlerType returns the identifier for this handler implementation.
func (h *ClaudeCodeAPIHandler) HandlerType() string {
	return Claude
}

// Models returns a list of models supported by this handler.
func (h *ClaudeCodeAPIHandler) Models() []map[string]any {
	// Get dynamic models from the global registry
	modelRegistry := registry.GetGlobalRegistry()
	return modelRegistry.GetAvailableModels("claude")
}

// ClaudeMessages handles Claude-compatible streaming chat completions.
// This function implements a sophisticated client rotation and quota management system
// to ensure high availability and optimal resource utilization across multiple backend clients.
//
// Parameters:
//   - c: The Gin context for the request.
func (h *ClaudeCodeAPIHandler) ClaudeMessages(c *gin.Context) {
	// Extract raw JSON data from the incoming request
	rawJSON, err := c.GetRawData()
	// If data retrieval fails, return a 400 Bad Request error.
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	// Check if the client requested a streaming response.
	streamResult := gjson.GetBytes(rawJSON, "stream")
	c.Set("claude_model", gjson.GetBytes(rawJSON, "model").String())
	c.Set("claude_provider", Claude)
	if !streamResult.Exists() || streamResult.Type == gjson.False {
		h.handleNonStreamingResponse(c, rawJSON)
	} else {
		h.handleStreamingResponse(c, rawJSON)
	}
}

// ClaudeMessages handles Claude-compatible streaming chat completions.
// This function implements a sophisticated client rotation and quota management system
// to ensure high availability and optimal resource utilization across multiple backend clients.
//
// Parameters:
//   - c: The Gin context for the request.
func (h *ClaudeCodeAPIHandler) ClaudeCountTokens(c *gin.Context) {
	// Extract raw JSON data from the incoming request
	rawJSON, err := c.GetRawData()
	// If data retrieval fails, return a 400 Bad Request error.
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	c.Header("Content-Type", "application/json")

	alt := h.GetAlt(c)
	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())

	modelName := gjson.GetBytes(rawJSON, "model").String()
	c.Set("claude_model", modelName)
	c.Set("claude_provider", Claude)

	resp, upstreamHeaders, errMsg := h.ExecuteCountWithAuthManager(cliCtx, h.HandlerType(), modelName, rawJSON, alt)
	if errMsg != nil {
		h.writeClaudeErrorResponse(c, errMsg)
		cliCancel(errMsg.Error)
		return
	}
	handlers.WriteUpstreamHeaders(c.Writer.Header(), upstreamHeaders)
	_, _ = c.Writer.Write(resp)
	cliCancel()
}

// ClaudeModels handles the Claude models listing endpoint.
// It returns a JSON response containing available Claude models and their specifications.
//
// Parameters:
//   - c: The Gin context for the request.
func (h *ClaudeCodeAPIHandler) ClaudeModels(c *gin.Context) {
	models := h.Models()
	firstID := ""
	lastID := ""
	if len(models) > 0 {
		if id, ok := models[0]["id"].(string); ok {
			firstID = id
		}
		if id, ok := models[len(models)-1]["id"].(string); ok {
			lastID = id
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     models,
		"has_more": false,
		"first_id": firstID,
		"last_id":  lastID,
	})
}

// handleNonStreamingResponse handles non-streaming content generation requests for Claude models.
// This function processes the request synchronously and returns the complete generated
// response in a single API call. It supports various generation parameters and
// response formats.
//
// Parameters:
//   - c: The Gin context for the request
//   - modelName: The name of the Gemini model to use for content generation
//   - rawJSON: The raw JSON request body containing generation parameters and content
func (h *ClaudeCodeAPIHandler) handleNonStreamingResponse(c *gin.Context, rawJSON []byte) {
	c.Header("Content-Type", "application/json")
	alt := h.GetAlt(c)
	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
	stopKeepAlive := h.StartNonStreamingKeepAlive(c, cliCtx)

	modelName := gjson.GetBytes(rawJSON, "model").String()
	c.Set("claude_model", modelName)
	c.Set("claude_provider", Claude)

	resp, upstreamHeaders, errMsg := h.ExecuteWithAuthManager(cliCtx, h.HandlerType(), modelName, rawJSON, alt)
	stopKeepAlive()
	if errMsg != nil {
		h.writeClaudeErrorResponse(c, errMsg)
		cliCancel(errMsg.Error)
		return
	}

	// Decompress gzipped responses - Claude API sometimes returns gzip without Content-Encoding header
	// This fixes title generation and other non-streaming responses that arrive compressed
	if len(resp) >= 2 && resp[0] == 0x1f && resp[1] == 0x8b {
		gzReader, errGzip := gzip.NewReader(bytes.NewReader(resp))
		if errGzip != nil {
			log.Warnf("failed to decompress gzipped Claude response: %v", errGzip)
		} else {
			defer func() {
				if errClose := gzReader.Close(); errClose != nil {
					log.Warnf("failed to close Claude gzip reader: %v", errClose)
				}
			}()
			decompressed, errRead := io.ReadAll(gzReader)
			if errRead != nil {
				log.Warnf("failed to read decompressed Claude response: %v", errRead)
			} else {
				resp = decompressed
			}
		}
	}

	handlers.WriteUpstreamHeaders(c.Writer.Header(), upstreamHeaders)
	_, _ = c.Writer.Write(resp)
	cliCancel()
}

// handleStreamingResponse streams Claude-compatible responses backed by Gemini.
// It sets up SSE, selects a backend client with rotation/quota logic,
// forwards chunks, and translates them to Claude CLI format.
//
// Parameters:
//   - c: The Gin context for the request.
//   - rawJSON: The raw JSON request body.
func (h *ClaudeCodeAPIHandler) handleStreamingResponse(c *gin.Context, rawJSON []byte) {
	// Get the http.Flusher interface to manually flush the response.
	// This is crucial for streaming as it allows immediate sending of data chunks
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: "Streaming not supported",
				Type:    "server_error",
			},
		})
		return
	}

	modelName := gjson.GetBytes(rawJSON, "model").String()
	c.Set("claude_model", modelName)
	c.Set("claude_provider", Claude)

	// Create a cancellable context for the backend client request
	// This allows proper cleanup and cancellation of ongoing requests
	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())

	dataChan, upstreamHeaders, errChan := h.ExecuteStreamWithAuthManager(cliCtx, h.HandlerType(), modelName, rawJSON, "")
	setSSEHeaders := func() {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")
	}

	// Peek at the first chunk to determine success or failure before setting headers
	for {
		select {
		case <-c.Request.Context().Done():
			cliCancel(c.Request.Context().Err())
			return
		case errMsg, ok := <-errChan:
			if !ok {
				// Err channel closed cleanly; wait for data channel.
				errChan = nil
				continue
			}
			// Upstream failed immediately. Return proper error status and JSON.
			h.writeClaudeErrorResponse(c, errMsg)
			if errMsg != nil {
				cliCancel(errMsg.Error)
			} else {
				cliCancel(nil)
			}
			return
		case chunk, ok := <-dataChan:
			if !ok {
				// Stream closed without data? Send DONE or just headers.
				setSSEHeaders()
				handlers.WriteUpstreamHeaders(c.Writer.Header(), upstreamHeaders)
				flusher.Flush()
				cliCancel(nil)
				return
			}

			// Success! Set headers now.
			setSSEHeaders()
			handlers.WriteUpstreamHeaders(c.Writer.Header(), upstreamHeaders)

			// Write the first chunk
			if len(chunk) > 0 {
				_, _ = c.Writer.Write(chunk)
				flusher.Flush()
			}

			// Continue streaming the rest
			h.forwardClaudeStream(c, flusher, func(err error) { cliCancel(err) }, dataChan, errChan)
			return
		}
	}
}

func (h *ClaudeCodeAPIHandler) forwardClaudeStream(c *gin.Context, flusher http.Flusher, cancel func(error), data <-chan []byte, errs <-chan *interfaces.ErrorMessage) {
	h.ForwardStream(c, flusher, cancel, data, errs, handlers.StreamForwardOptions{
		WriteChunk: func(chunk []byte) {
			if len(chunk) == 0 {
				return
			}
			_, _ = c.Writer.Write(chunk)
		},
		WriteTerminalError: func(errMsg *interfaces.ErrorMessage) {
			if errMsg == nil {
				return
			}
			status := http.StatusInternalServerError
			if errMsg.StatusCode > 0 {
				status = errMsg.StatusCode
			}
			c.Status(status)

			errorBytes, _ := json.Marshal(h.toClaudeError(errMsg, true))
			_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorBytes)
		},
	})
}

type claudeErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type claudeErrorResponse struct {
	Type  string            `json:"type"`
	Error claudeErrorDetail `json:"error"`
}

func (h *ClaudeCodeAPIHandler) toClaudeError(msg *interfaces.ErrorMessage, includeRetryHint bool) claudeErrorResponse {
	status := http.StatusInternalServerError
	if msg != nil && msg.StatusCode > 0 {
		status = msg.StatusCode
	}
	message := claudeErrorMessage(msg, status)
	if includeRetryHint {
		message = appendClaudeRetryHint(message, msg)
	}
	return claudeErrorResponse{
		Type: "error",
		Error: claudeErrorDetail{
			Type:    claudeErrorType(msg, status),
			Message: message,
		},
	}
}

func (h *ClaudeCodeAPIHandler) writeClaudeErrorResponse(c *gin.Context, msg *interfaces.ErrorMessage) {
	status := http.StatusInternalServerError
	if msg != nil && msg.StatusCode > 0 {
		status = msg.StatusCode
	}
	if msg != nil && msg.Addon != nil {
		handlers.WriteErrorResponseHeaders(c.Writer.Header(), msg.Addon, handlers.PassthroughHeadersEnabled(h.Cfg))
	}

	payload := h.toClaudeError(msg, false)
	body, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		body = []byte(`{"type":"error","error":{"type":"api_error","message":"failed to encode Claude error response"}}`)
	}
	c.Set("API_RESPONSE", body)
	if !c.Writer.Written() {
		c.Writer.Header().Set("Content-Type", "application/json")
	}
	c.Status(status)
	_, _ = c.Writer.Write(body)
	h.logClaudeRateLimitCompatibility(c, msg, payload)
}

func claudeErrorType(msg *interfaces.ErrorMessage, status int) string {
	if msg != nil && msg.Error != nil {
		if details, ok := coreauth.CooldownDetailsFromError(msg.Error); ok && strings.EqualFold(details.Reason, "quota") {
			return "rate_limit_error"
		}
	}

	switch status {
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity:
		return "invalid_request_error"
	default:
		return "api_error"
	}
}

func claudeErrorMessage(msg *interfaces.ErrorMessage, status int) string {
	if msg == nil || msg.Error == nil {
		return http.StatusText(status)
	}
	if details, ok := coreauth.CooldownDetailsFromError(msg.Error); ok {
		if extracted := extractClaudeErrorMessage(details.SourceMessage); extracted != "" {
			return extracted
		}
	}

	var authErr *coreauth.Error
	if errors.As(msg.Error, &authErr) && authErr != nil {
		if extracted := extractClaudeErrorMessage(authErr.Message); extracted != "" {
			return extracted
		}
		if trimmed := strings.TrimSpace(authErr.Message); trimmed != "" {
			return trimmed
		}
	}

	if extracted := extractClaudeErrorMessage(msg.Error.Error()); extracted != "" {
		return extracted
	}
	return http.StatusText(status)
}

func extractClaudeErrorMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if json.Valid([]byte(raw)) {
		if message := strings.TrimSpace(gjson.Get(raw, "error.message").String()); message != "" {
			return message
		}
		if message := strings.TrimSpace(gjson.Get(raw, "message").String()); message != "" {
			return message
		}
	}
	return raw
}

func appendClaudeRetryHint(message string, msg *interfaces.ErrorMessage) string {
	retryHint := claudeRetryHint(msg)
	if retryHint == "" {
		return message
	}
	if strings.Contains(message, retryHint) {
		return message
	}
	if message == "" {
		return retryHint
	}
	return fmt.Sprintf("%s %s", message, retryHint)
}

func claudeRetryHint(msg *interfaces.ErrorMessage) string {
	if msg == nil || msg.Addon == nil {
		return ""
	}
	raw := strings.TrimSpace(msg.Addon.Get("Retry-After"))
	if raw == "" {
		return ""
	}
	if seconds, errParse := strconv.Atoi(raw); errParse == nil {
		unit := "seconds"
		if seconds == 1 {
			unit = "second"
		}
		return fmt.Sprintf("Retry after %d %s.", seconds, unit)
	}
	return fmt.Sprintf("Retry after %s.", raw)
}

func (h *ClaudeCodeAPIHandler) logClaudeRateLimitCompatibility(c *gin.Context, msg *interfaces.ErrorMessage, payload claudeErrorResponse) {
	if payload.Error.Type != "rate_limit_error" {
		return
	}
	source := "upstream_quota"
	provider := Claude
	model, _ := c.Get("claude_model")
	modelName, _ := model.(string)
	resetSeconds := 0

	if msg != nil && msg.Error != nil {
		if details, ok := coreauth.CooldownDetailsFromError(msg.Error); ok {
			source = "local_cooldown"
			if details.Provider != "" {
				provider = details.Provider
			}
			if details.Model != "" {
				modelName = details.Model
			}
			if details.ResetIn > 0 {
				resetSeconds = int(details.ResetIn.Round(time.Second) / time.Second)
			}
		}
	}
	if resetSeconds == 0 {
		if retryHint := claudeRetryHint(msg); retryHint != "" {
			if raw := strings.TrimSpace(msg.Addon.Get("Retry-After")); raw != "" {
				if seconds, errParse := strconv.Atoi(raw); errParse == nil {
					resetSeconds = seconds
				}
			}
		}
	}

	log.WithFields(log.Fields{
		"session_id":    strings.TrimSpace(c.GetHeader("X-Claude-Code-Session-Id")),
		"model":         modelName,
		"provider":      provider,
		"reset_seconds": resetSeconds,
		"compat_source": source,
		"error_type":    payload.Error.Type,
	}).Warn("claude compatible rate-limit error emitted")
}
