package auth

import (
	"errors"
	"strings"
	"time"
)

// CooldownDetails captures the structured reason behind a synthesized cooldown error.
type CooldownDetails struct {
	Reason        string
	SourceMessage string
	Provider      string
	Model         string
	ResetIn       time.Duration
}

// CooldownDetailsFromError extracts structured cooldown metadata from an error.
func CooldownDetailsFromError(err error) (CooldownDetails, bool) {
	if err == nil {
		return CooldownDetails{}, false
	}
	var cooldownErr *modelCooldownError
	if !errors.As(err, &cooldownErr) || cooldownErr == nil {
		return CooldownDetails{}, false
	}
	return cooldownErr.details(), true
}

func cooldownDetailsForAuth(auth *Auth, model string, now time.Time) (CooldownDetails, bool) {
	if auth == nil {
		return CooldownDetails{}, false
	}

	state, ok := cooldownStateForAuth(auth, model)
	if ok && state != nil {
		resetAt := cooldownResetAt(state.NextRetryAfter, state.Quota.NextRecoverAt, now)
		if state.Unavailable && !resetAt.IsZero() && state.Quota.Exceeded {
			return buildCooldownDetails(auth.Provider, model, resetAt.Sub(now), state.Quota.Reason, state.LastError, state.StatusMessage), true
		}
	}

	resetAt := cooldownResetAt(auth.NextRetryAfter, auth.Quota.NextRecoverAt, now)
	if auth.Unavailable && !resetAt.IsZero() && auth.Quota.Exceeded {
		return buildCooldownDetails(auth.Provider, model, resetAt.Sub(now), auth.Quota.Reason, auth.LastError, auth.StatusMessage), true
	}

	return CooldownDetails{}, false
}

func cooldownDetailsForAuths(auths []*Auth, model string, now time.Time) (CooldownDetails, bool) {
	chosen := CooldownDetails{}
	found := false
	for i := 0; i < len(auths); i++ {
		details, ok := cooldownDetailsForAuth(auths[i], model, now)
		if !ok {
			continue
		}
		if !found || shouldPreferCooldownDetails(details, chosen) {
			chosen = details
			found = true
		}
	}
	return chosen, found
}

func shouldPreferCooldownDetails(candidate, current CooldownDetails) bool {
	if current.ResetIn <= 0 {
		return candidate.ResetIn > 0
	}
	if candidate.ResetIn <= 0 {
		return false
	}
	return candidate.ResetIn < current.ResetIn
}

func cooldownStateForAuth(auth *Auth, model string) (*ModelState, bool) {
	if auth == nil || model == "" || len(auth.ModelStates) == 0 {
		return nil, false
	}
	if state, ok := auth.ModelStates[model]; ok && state != nil {
		return state, true
	}
	baseModel := canonicalModelKey(model)
	if baseModel != "" && baseModel != model {
		if state, ok := auth.ModelStates[baseModel]; ok && state != nil {
			return state, true
		}
	}
	return nil, false
}

func cooldownResetAt(nextRetryAfter, nextRecoverAt, now time.Time) time.Time {
	resetAt := nextRetryAfter
	if !nextRecoverAt.IsZero() && nextRecoverAt.After(now) {
		resetAt = nextRecoverAt
	}
	if resetAt.IsZero() || !resetAt.After(now) {
		return time.Time{}
	}
	return resetAt
}

func buildCooldownDetails(provider, model string, resetIn time.Duration, reason string, lastErr *Error, fallbackMessage string) CooldownDetails {
	if resetIn < 0 {
		resetIn = 0
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "quota"
	}

	sourceMessage := strings.TrimSpace(fallbackMessage)
	if lastErr != nil && strings.TrimSpace(lastErr.Message) != "" {
		sourceMessage = strings.TrimSpace(lastErr.Message)
	}

	return CooldownDetails{
		Reason:        reason,
		SourceMessage: sourceMessage,
		Provider:      strings.TrimSpace(provider),
		Model:         strings.TrimSpace(model),
		ResetIn:       resetIn,
	}
}
