package executor

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/tidwall/gjson"
)

func TestAntigravityBuildRequest_SanitizesGeminiToolSchema(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "gemini-2.5-pro")

	decl := extractFirstFunctionDeclaration(t, body)
	if _, ok := decl["parametersJsonSchema"]; ok {
		t.Fatalf("parametersJsonSchema should be renamed to parameters")
	}

	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("parameters missing or invalid type")
	}
	assertSchemaSanitizedAndPropertyPreserved(t, params)
}

func TestAntigravityBuildRequest_SanitizesAntigravityToolSchema(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "claude-opus-4-6")

	decl := extractFirstFunctionDeclaration(t, body)
	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("parameters missing or invalid type")
	}
	assertSchemaSanitizedAndPropertyPreserved(t, params)
}

func TestAntigravityBuildRequest_SkipsSchemaSanitizationWithoutToolsField(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-flash-image", []byte(`{
		"request": {
			"contents": [
				{
					"role": "user",
					"x-debug": "keep-me",
					"parts": [
						{
							"text": "hello"
						}
					]
				}
			],
			"nonSchema": {
				"nullable": true,
				"x-extra": "keep-me"
			},
			"generationConfig": {
				"maxOutputTokens": 128
			}
		}
	}`))

	assertNonSchemaRequestPreserved(t, body)
}

func TestAntigravityBuildRequest_SkipsSchemaSanitizationWithEmptyToolsArray(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-flash-image", []byte(`{
		"request": {
			"tools": [],
			"contents": [
				{
					"role": "user",
					"x-debug": "keep-me",
					"parts": [
						{
							"text": "hello"
						}
					]
				}
			],
			"nonSchema": {
				"nullable": true,
				"x-extra": "keep-me"
			},
			"generationConfig": {
				"maxOutputTokens": 128
			}
		}
	}`))

	assertNonSchemaRequestPreserved(t, body)
}

func TestAntigravityBuildRequest_PreservesCacheControlAndAliasedToolNames(t *testing.T) {
	aliasName := "ag_browser_subagent__0123456789abcdef"
	body := buildRequestBodyFromRawPayload(t, "claude-opus-4-6", []byte(`{
		"request": {
			"systemInstruction": {
				"parts": [
					{"text": "system", "cache_control": {"type": "ephemeral"}}
				]
			},
			"contents": [
				{
					"role": "user",
					"parts": [
						{"text": "hello", "cache_control": {"type": "ephemeral"}}
					]
				}
			],
			"tools": [
				{
					"function_declarations": [
						{
							"name": "`+aliasName+`",
							"cache_control": {"type": "ephemeral"},
							"parametersJsonSchema": {
								"type": "object",
								"properties": {"url": {"type": "string"}}
							}
						}
					]
				}
			],
			"toolConfig": {
				"functionCallingConfig": {
					"mode": "ANY",
					"allowedFunctionNames": ["`+aliasName+`"]
				}
			}
		}
	}`))

	decl := extractFirstFunctionDeclaration(t, body)
	if got, _ := decl["name"].(string); got != aliasName {
		t.Fatalf("aliased tool name should be preserved, got %v", decl["name"])
	}
	cacheControl, ok := decl["cache_control"].(map[string]any)
	if !ok || cacheControl["type"] != "ephemeral" {
		t.Fatalf("tool declaration cache_control should be preserved, got %v", decl["cache_control"])
	}

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}
	systemInstruction, ok := request["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatalf("systemInstruction missing or invalid type")
	}
	systemParts, ok := systemInstruction["parts"].([]any)
	if !ok || len(systemParts) != 1 {
		t.Fatalf("systemInstruction.parts missing or invalid type")
	}
	systemPart, ok := systemParts[0].(map[string]any)
	if !ok {
		t.Fatalf("system part missing or invalid type")
	}
	systemCacheControl, ok := systemPart["cache_control"].(map[string]any)
	if !ok || systemCacheControl["type"] != "ephemeral" {
		t.Fatalf("system cache_control should be preserved, got %v", systemPart["cache_control"])
	}

	contents, ok := request["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents missing or invalid type")
	}
	content, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content missing or invalid type")
	}
	parts, ok := content["parts"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("content parts missing or invalid type")
	}
	contentPart, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("content part missing or invalid type")
	}
	contentCacheControl, ok := contentPart["cache_control"].(map[string]any)
	if !ok || contentCacheControl["type"] != "ephemeral" {
		t.Fatalf("content cache_control should be preserved, got %v", contentPart["cache_control"])
	}

	toolConfig, ok := request["toolConfig"].(map[string]any)
	if !ok {
		t.Fatalf("toolConfig missing or invalid type")
	}
	functionCallingConfig, ok := toolConfig["functionCallingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("functionCallingConfig missing or invalid type")
	}
	allowedFunctionNames, ok := functionCallingConfig["allowedFunctionNames"].([]any)
	if !ok || len(allowedFunctionNames) != 1 || allowedFunctionNames[0] != aliasName {
		t.Fatalf("allowedFunctionNames should be preserved, got %v", functionCallingConfig["allowedFunctionNames"])
	}
}

func TestAntigravityConvertStreamToNonStream_PreservesFunctionCallID(t *testing.T) {
	executor := &AntigravityExecutor{}
	stream := []byte(`{"response":{"responseId":"resp_1","modelVersion":"claude-sonnet-4-6","candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call_123","name":"ag_tool__0123456789abcdef","args":{"path":"."}}}]}}]}}`)

	output := executor.convertStreamToNonStream(append(stream, '\n'))

	if got := gjson.GetBytes(output, "response.candidates.0.content.parts.0.functionCall.id").String(); got != "call_123" {
		t.Fatalf("functionCall.id should survive stream-to-nonstream conversion, got %q", got)
	}
}

func assertNonSchemaRequestPreserved(t *testing.T, body map[string]any) {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}

	contents, ok := request["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("contents missing or empty")
	}
	content, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content missing or invalid type")
	}
	if got, ok := content["x-debug"].(string); !ok || got != "keep-me" {
		t.Fatalf("x-debug should be preserved when no tool schema exists, got=%v", content["x-debug"])
	}

	nonSchema, ok := request["nonSchema"].(map[string]any)
	if !ok {
		t.Fatalf("nonSchema missing or invalid type")
	}
	if _, ok := nonSchema["nullable"]; !ok {
		t.Fatalf("nullable should be preserved outside schema cleanup path")
	}
	if got, ok := nonSchema["x-extra"].(string); !ok || got != "keep-me" {
		t.Fatalf("x-extra should be preserved outside schema cleanup path, got=%v", nonSchema["x-extra"])
	}

	if generationConfig, ok := request["generationConfig"].(map[string]any); ok {
		if _, ok := generationConfig["maxOutputTokens"]; ok {
			t.Fatalf("maxOutputTokens should still be removed for non-Claude requests")
		}
	}
}

func buildRequestBodyFromPayload(t *testing.T, modelName string) map[string]any {
	t.Helper()
	return buildRequestBodyFromRawPayload(t, modelName, []byte(`{
		"request": {
			"tools": [
				{
					"function_declarations": [
						{
							"name": "tool_1",
							"parametersJsonSchema": {
								"$schema": "http://json-schema.org/draft-07/schema#",
								"$id": "root-schema",
								"type": "object",
								"properties": {
									"$id": {"type": "string"},
									"arg": {
										"type": "object",
										"prefill": "hello",
										"properties": {
											"mode": {
												"type": "string",
												"deprecated": true,
												"enum": ["a", "b"],
												"enumTitles": ["A", "B"]
											}
										}
									}
								},
								"patternProperties": {
									"^x-": {"type": "string"}
								}
							}
						}
					]
				}
			]
		}
	}`))
}

func buildRequestBodyFromRawPayload(t *testing.T, modelName string, payload []byte) map[string]any {
	t.Helper()

	executor := &AntigravityExecutor{}
	auth := &cliproxyauth.Auth{}

	req, err := executor.buildRequest(context.Background(), auth, "token", modelName, payload, false, "", "https://example.com")
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal request body error: %v, body=%s", err, string(raw))
	}
	return body
}

func extractFirstFunctionDeclaration(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}
	tools, ok := request["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools missing or empty")
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("first tool invalid type")
	}
	decls, ok := tool["function_declarations"].([]any)
	if !ok || len(decls) == 0 {
		t.Fatalf("function_declarations missing or empty")
	}
	decl, ok := decls[0].(map[string]any)
	if !ok {
		t.Fatalf("first function declaration invalid type")
	}
	return decl
}

func assertSchemaSanitizedAndPropertyPreserved(t *testing.T, params map[string]any) {
	t.Helper()

	if _, ok := params["$id"]; ok {
		t.Fatalf("root $id should be removed from schema")
	}
	if _, ok := params["patternProperties"]; ok {
		t.Fatalf("patternProperties should be removed from schema")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or invalid type")
	}
	if _, ok := props["$id"]; !ok {
		t.Fatalf("property named $id should be preserved")
	}

	arg, ok := props["arg"].(map[string]any)
	if !ok {
		t.Fatalf("arg property missing or invalid type")
	}
	if _, ok := arg["prefill"]; ok {
		t.Fatalf("prefill should be removed from nested schema")
	}

	argProps, ok := arg["properties"].(map[string]any)
	if !ok {
		t.Fatalf("arg.properties missing or invalid type")
	}
	mode, ok := argProps["mode"].(map[string]any)
	if !ok {
		t.Fatalf("mode property missing or invalid type")
	}
	if _, ok := mode["enumTitles"]; ok {
		t.Fatalf("enumTitles should be removed from nested schema")
	}
	if _, ok := mode["deprecated"]; ok {
		t.Fatalf("deprecated should be removed from nested schema")
	}
}
