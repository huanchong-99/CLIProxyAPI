package synthesizer

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestConfigSynthesizer_ZhipuKeys(t *testing.T) {
	synth := NewConfigSynthesizer()
	ctx := &SynthesisContext{
		Config: &config.Config{
			ZhipuAPIKey: []config.ZhipuKey{
				{
					APIKey:   "zhipu-key",
					Prefix:   "team-z",
					ProxyURL: "http://proxy.local:8080/path",
					Models: []config.ZhipuModel{
						{Name: "glm-4.6", Alias: "glm-god"},
					},
					Headers: map[string]string{"X-Zhipu": "value"},
					ExcludedModels: []string{
						"glm-hidden",
					},
				},
			},
		},
		Now:         time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		IDGenerator: NewStableIDGenerator(),
	}

	auths, err := synth.Synthesize(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auths) != 1 {
		t.Fatalf("expected 1 auth, got %d", len(auths))
	}

	auth := auths[0]
	if auth.Provider != "zhipu" {
		t.Fatalf("provider = %q, want %q", auth.Provider, "zhipu")
	}
	if auth.Label != "zhipu-apikey" {
		t.Fatalf("label = %q, want %q", auth.Label, "zhipu-apikey")
	}
	if auth.Status != coreauth.StatusActive {
		t.Fatalf("status = %q, want %q", auth.Status, coreauth.StatusActive)
	}
	if auth.Prefix != "team-z" {
		t.Fatalf("prefix = %q, want %q", auth.Prefix, "team-z")
	}
	if auth.ProxyURL != "http://proxy.local:8080/path" {
		t.Fatalf("proxy_url = %q, want %q", auth.ProxyURL, "http://proxy.local:8080/path")
	}
	if got := auth.Attributes["api_key"]; got != "zhipu-key" {
		t.Fatalf("api_key = %q, want %q", got, "zhipu-key")
	}
	if got := auth.Attributes["base_url"]; got != config.DefaultZhipuBaseURL {
		t.Fatalf("base_url = %q, want %q", got, config.DefaultZhipuBaseURL)
	}
	if got := auth.Attributes["provider_key"]; got != "zhipu" {
		t.Fatalf("provider_key = %q, want %q", got, "zhipu")
	}
	if got := auth.Attributes["header:X-Zhipu"]; got != "value" {
		t.Fatalf("header:X-Zhipu = %q, want %q", got, "value")
	}
	if got := auth.Attributes["excluded_models"]; got != "glm-hidden" {
		t.Fatalf("excluded_models = %q, want %q", got, "glm-hidden")
	}
	if got := auth.Attributes["auth_kind"]; got != "apikey" {
		t.Fatalf("auth_kind = %q, want %q", got, "apikey")
	}
	if got := auth.Attributes["source"]; got == "" {
		t.Fatal("expected non-empty source attribute")
	}
	if got := auth.Attributes["models_hash"]; got == "" {
		t.Fatal("expected non-empty models_hash attribute")
	}
	if got := auth.Attributes["excluded_models_hash"]; got == "" {
		t.Fatal("expected non-empty excluded_models_hash attribute")
	}
}

func TestConfigSynthesizer_ZhipuKeys_UsesLegacyStableIDAndSkipsLegacyCompatDuplicate(t *testing.T) {
	cfg := &config.Config{
		OpenAICompatibility: []config.OpenAICompatibility{
			{
				Name:    "zhipu",
				BaseURL: config.DefaultZhipuBaseURL,
				APIKeyEntries: []config.OpenAICompatibilityAPIKey{
					{
						APIKey:   "legacy-zhipu-key",
						ProxyURL: "socks5://proxy.example.com:1080",
					},
				},
				Models: []config.OpenAICompatibilityModel{
					{Name: "glm-4.6", Alias: "glm-god"},
				},
			},
		},
	}
	cfg.SynthesizeLegacyZhipuKeys()
	cfg.SanitizeZhipuKeys()

	gen := NewStableIDGenerator()
	wantID, _ := gen.Next("openai-compatibility:zhipu", "legacy-zhipu-key", config.DefaultZhipuBaseURL, "socks5://proxy.example.com:1080")

	auths, err := NewConfigSynthesizer().Synthesize(&SynthesisContext{
		Config:      cfg,
		Now:         time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		IDGenerator: NewStableIDGenerator(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auths) != 1 {
		t.Fatalf("expected 1 synthesized auth, got %d", len(auths))
	}
	if auths[0].ID != wantID {
		t.Fatalf("id = %q, want %q", auths[0].ID, wantID)
	}
	if auths[0].Provider != "zhipu" {
		t.Fatalf("provider = %q, want %q", auths[0].Provider, "zhipu")
	}
}

func TestConfigSynthesizer_ZhipuKeys_IDMatchesEquivalentLegacyConfig(t *testing.T) {
	now := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	firstClass := &config.Config{
		ZhipuAPIKey: []config.ZhipuKey{
			{
				APIKey:   "shared-key",
				BaseURL:  config.DefaultZhipuBaseURL,
				ProxyURL: "direct",
			},
		},
	}
	legacy := &config.Config{
		OpenAICompatibility: []config.OpenAICompatibility{
			{
				Name:    "zhipu",
				BaseURL: config.DefaultZhipuBaseURL,
				APIKeyEntries: []config.OpenAICompatibilityAPIKey{
					{
						APIKey:   "shared-key",
						ProxyURL: "direct",
					},
				},
			},
		},
	}
	legacy.SynthesizeLegacyZhipuKeys()
	legacy.SanitizeZhipuKeys()

	synth := NewConfigSynthesizer()
	firstAuths, err := synth.Synthesize(&SynthesisContext{
		Config:      firstClass,
		Now:         now,
		IDGenerator: NewStableIDGenerator(),
	})
	if err != nil {
		t.Fatalf("unexpected error for first-class config: %v", err)
	}
	legacyAuths, err := synth.Synthesize(&SynthesisContext{
		Config:      legacy,
		Now:         now,
		IDGenerator: NewStableIDGenerator(),
	})
	if err != nil {
		t.Fatalf("unexpected error for legacy config: %v", err)
	}
	if len(firstAuths) != 1 || len(legacyAuths) != 1 {
		t.Fatalf("expected one auth from each config, got %d and %d", len(firstAuths), len(legacyAuths))
	}
	if firstAuths[0].ID != legacyAuths[0].ID {
		t.Fatalf("ids differ: first-class=%q legacy=%q", firstAuths[0].ID, legacyAuths[0].ID)
	}
}
