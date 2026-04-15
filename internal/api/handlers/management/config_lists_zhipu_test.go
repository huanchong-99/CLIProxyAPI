package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestPutZhipuKeys_ReplacesConfiguredEntries(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	body, err := json.Marshal([]config.ZhipuKey{{
		APIKey: "zhipu-key",
		Models: []config.ZhipuModel{{Name: "glm-4.5", Alias: "glm-4.5"}},
	}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/v0/management/zhipu-api-key", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PutZhipuKeys(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := len(h.cfg.ZhipuAPIKey); got != 1 {
		t.Fatalf("zhipu keys len = %d, want 1", got)
	}
	if got := h.cfg.ZhipuAPIKey[0].BaseURL; got != config.DefaultZhipuBaseURL {
		t.Fatalf("base-url = %q, want %q", got, config.DefaultZhipuBaseURL)
	}
}

func TestPatchZhipuKey_UpdatesEntryByIndex(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg: &config.Config{
			ZhipuAPIKey: []config.ZhipuKey{{
				APIKey:  "zhipu-key",
				BaseURL: config.DefaultZhipuBaseURL,
				Models:  []config.ZhipuModel{{Name: "glm-4.5", Alias: "glm-4.5"}},
			}},
		},
		configFilePath: writeTestConfigFile(t),
	}

	body := []byte(`{"index":0,"value":{"proxy-url":"http://127.0.0.1:8080"}}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/zhipu-api-key", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.PatchZhipuKey(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := h.cfg.ZhipuAPIKey[0].ProxyURL; got != "http://127.0.0.1:8080" {
		t.Fatalf("proxy-url = %q, want %q", got, "http://127.0.0.1:8080")
	}
}

func TestDeleteZhipuKey_DeletesOnlyMatchingBaseURL(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	h := &Handler{
		cfg: &config.Config{
			ZhipuAPIKey: []config.ZhipuKey{
				{APIKey: "shared-key", BaseURL: config.DefaultZhipuBaseURL},
				{APIKey: "shared-key", BaseURL: "https://custom.example.com/anthropic"},
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/zhipu-api-key?api-key=shared-key&base-url=https://custom.example.com/anthropic", nil)

	h.DeleteZhipuKey(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := len(h.cfg.ZhipuAPIKey); got != 1 {
		t.Fatalf("zhipu keys len = %d, want 1", got)
	}
	if got := h.cfg.ZhipuAPIKey[0].BaseURL; got != config.DefaultZhipuBaseURL {
		t.Fatalf("remaining base-url = %q, want %q", got, config.DefaultZhipuBaseURL)
	}
}
