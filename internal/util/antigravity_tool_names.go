package util

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	antigravityToolNameMaxLen = 64
	antigravityToolPrefix     = "ag_"
	antigravityToolSeparator  = "__"
)

var antigravityToolStemSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.:-]`)

// AntigravityToolName returns a collision-resistant tool alias for the
// Antigravity Claude compatibility path. Names that already satisfy Gemini's
// function-name requirements and length limit are preserved unchanged.
func AntigravityToolName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) <= antigravityToolNameMaxLen && SanitizeFunctionName(trimmed) == trimmed {
		return trimmed
	}

	stem := antigravityToolStemSanitizer.ReplaceAllString(trimmed, "_")
	stem = strings.Trim(stem, "_.:-")
	stem = strings.TrimLeft(stem, "0123456789")
	if stem == "" {
		stem = "tool"
	}

	hash := sha256.Sum256([]byte(trimmed))
	suffix := hex.EncodeToString(hash[:8])
	maxStemLen := antigravityToolNameMaxLen - len(antigravityToolPrefix) - len(antigravityToolSeparator) - len(suffix)
	if maxStemLen < 1 {
		maxStemLen = 1
	}
	if len(stem) > maxStemLen {
		stem = stem[:maxStemLen]
	}

	return antigravityToolPrefix + stem + antigravityToolSeparator + suffix
}

// AntigravityToolNameMap builds an aliased-name -> original-name map from a
// Claude request so downstream response translators can restore the original
// client-facing tool name after Antigravity aliasing.
func AntigravityToolNameMap(rawJSON []byte) map[string]string {
	if len(rawJSON) == 0 || !gjson.ValidBytes(rawJSON) {
		return nil
	}

	tools := gjson.GetBytes(rawJSON, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return nil
	}

	out := make(map[string]string)
	tools.ForEach(func(_, tool gjson.Result) bool {
		name := strings.TrimSpace(tool.Get("name").String())
		if name == "" {
			return true
		}

		aliased := AntigravityToolName(name)
		if aliased == name {
			return true
		}

		if existing, ok := out[aliased]; ok && existing != name {
			log.Warnf("antigravity tool alias collision: %q and %q both map to %q, keeping first", existing, name, aliased)
			return true
		}
		out[aliased] = name
		return true
	})

	if len(out) == 0 {
		return nil
	}
	return out
}
