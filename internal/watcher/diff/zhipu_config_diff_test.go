package diff

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestBuildConfigChangeDetails_ZhipuKeyCountChanges(t *testing.T) {
	oldCfg := &config.Config{
		ZhipuAPIKey: []config.ZhipuKey{
			{APIKey: "z1", BaseURL: config.DefaultZhipuBaseURL},
		},
	}
	newCfg := &config.Config{
		ZhipuAPIKey: []config.ZhipuKey{
			{APIKey: "z1", BaseURL: config.DefaultZhipuBaseURL},
			{APIKey: "z2", BaseURL: config.DefaultZhipuBaseURL},
		},
	}

	changes := BuildConfigChangeDetails(oldCfg, newCfg)
	expectContains(t, changes, "zhipu-api-key count: 1 -> 2")
}

func TestBuildConfigChangeDetails_ZhipuKeyDetailsAreRedacted(t *testing.T) {
	oldCfg := &config.Config{
		ZhipuAPIKey: []config.ZhipuKey{
			{
				APIKey:   "old-secret",
				Prefix:   "old-team",
				BaseURL:  config.DefaultZhipuBaseURL,
				ProxyURL: "http://old-proxy.local:8080/path",
				Models: []config.ZhipuModel{
					{Name: "glm-4.6", Alias: "glm-god"},
				},
				Headers: map[string]string{"X-Zhipu": "old"},
				ExcludedModels: []string{
					"old-hidden",
				},
			},
		},
	}
	newCfg := &config.Config{
		ZhipuAPIKey: []config.ZhipuKey{
			{
				APIKey:   "new-secret",
				Prefix:   "new-team",
				BaseURL:  "https://open.bigmodel.cn/api/anthropic/v2",
				ProxyURL: "http://new-proxy.local:8081/path",
				Models: []config.ZhipuModel{
					{Name: "glm-4.6", Alias: "glm-god"},
					{Name: "glm-4.7", Alias: "glm-god"},
				},
				Headers: map[string]string{"X-Zhipu": "new"},
				ExcludedModels: []string{
					"old-hidden",
					"new-hidden",
				},
			},
		},
	}

	changes := BuildConfigChangeDetails(oldCfg, newCfg)
	expectContains(t, changes, "zhipu[0].base-url: https://open.bigmodel.cn/api/anthropic -> https://open.bigmodel.cn/api/anthropic/v2")
	expectContains(t, changes, "zhipu[0].proxy-url: http://old-proxy.local:8080 -> http://new-proxy.local:8081")
	expectContains(t, changes, "zhipu[0].prefix: old-team -> new-team")
	expectContains(t, changes, "zhipu[0].api-key: updated")
	expectContains(t, changes, "zhipu[0].headers: updated")
	expectContains(t, changes, "zhipu[0].models: updated (1 -> 2 entries)")
	expectContains(t, changes, "zhipu[0].excluded-models: updated (1 -> 2 entries)")
}
