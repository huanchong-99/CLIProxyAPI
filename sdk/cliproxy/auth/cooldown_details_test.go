package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

func TestSelectorPick_AllCooldownErrorPreservesQuotaDetails(t *testing.T) {
	t.Parallel()

	model := "claude-sonnet-4-6"
	next := time.Now().Add(30 * time.Second)
	auths := []*Auth{
		{
			ID:       "auth-a",
			Provider: "claude",
			ModelStates: map[string]*ModelState{
				model: {
					Status:         StatusError,
					Unavailable:    true,
					NextRetryAfter: next,
					LastError: &Error{
						HTTPStatus: http.StatusTooManyRequests,
						Message:    `{"error":{"message":"Resource has been exhausted (e.g. check quota)."}}`,
					},
					Quota: QuotaState{
						Exceeded:      true,
						Reason:        "quota",
						NextRecoverAt: next,
					},
				},
			},
		},
	}

	selector := &FillFirstSelector{}
	_, err := selector.Pick(context.Background(), "claude", model, cliproxyexecutor.Options{}, auths)
	if err == nil {
		t.Fatalf("Pick() error = nil")
	}

	details, ok := CooldownDetailsFromError(err)
	if !ok {
		t.Fatalf("CooldownDetailsFromError() = false")
	}
	if details.Reason != "quota" {
		t.Fatalf("details.Reason = %q, want %q", details.Reason, "quota")
	}
	if details.Provider != "claude" {
		t.Fatalf("details.Provider = %q, want %q", details.Provider, "claude")
	}
	if details.Model != model {
		t.Fatalf("details.Model = %q, want %q", details.Model, model)
	}
	if details.ResetIn <= 0 {
		t.Fatalf("details.ResetIn = %v, want > 0", details.ResetIn)
	}
	if !strings.Contains(details.SourceMessage, "Resource has been exhausted") {
		t.Fatalf("details.SourceMessage = %q, want upstream quota message", details.SourceMessage)
	}

	var cooldownErr *modelCooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("Pick() error = %T, want *modelCooldownError", err)
	}
	if !strings.Contains(cooldownErr.Error(), `"reason":"quota"`) {
		t.Fatalf("Error() missing reason field: %s", cooldownErr.Error())
	}
	if !strings.Contains(cooldownErr.Error(), `"source_message"`) {
		t.Fatalf("Error() missing source_message field: %s", cooldownErr.Error())
	}
}
