package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kavix/why/internal/model"
)

// Provider defines the interface for an AI causal analysis backend.
type Provider interface {
	Name() string
	Available() bool
	Analyze(ctx context.Context, diag *model.Diagnostic) (string, error)
}

// Engine coordinates AI-powered root cause analysis and remediation suggestions.
type Engine struct {
	providers []Provider
}

func New() *Engine {
	return &Engine{
		providers: []Provider{
			&OllamaProvider{},
			&GeminiProvider{},
			&OpenAIProvider{},
			&AnthropicProvider{},
		},
	}
}

// Analyze feeds diagnostic evidence to an available AI provider to generate deep insights.
func (e *Engine) Analyze(ctx context.Context, diag *model.Diagnostic) (string, error) {
	for _, p := range e.providers {
		if p.Available() {
			analysis, err := p.Analyze(ctx, diag)
			if err == nil && len(strings.TrimSpace(analysis)) > 0 {
				return fmt.Sprintf("AI Root Cause Analysis (%s):\n\n%s", p.Name(), strings.TrimSpace(analysis)), nil
			}
		}
	}

	return "", fmt.Errorf(`No AI backend configured or reachable.

To enable AI-assisted causal troubleshooting, configure one of the following:

  1. Ollama (Local & Offline - Recommended):
     Run: 'ollama run llama3.2' (starts local daemon at localhost:11434)

  2. Google Gemini:
     export GEMINI_API_KEY="your-gemini-key"

  3. OpenAI / Compatible (vLLM, LocalAI):
     export OPENAI_API_KEY="your-openai-key"
     (Optional: export OPENAI_BASE_URL="http://localhost:8000/v1")

  4. Anthropic Claude:
     export ANTHROPIC_API_KEY="your-anthropic-key"
`)
}

// BuildPrompt constructs a high-density diagnostic prompt from structured facts.
func buildPrompt(diag *model.Diagnostic) string {
	jsonBytes, _ := json.MarshalIndent(diag, "", "  ")

	return fmt.Sprintf(`You are the causal diagnosis engine of "why", a Unix-native system troubleshooting tool.
Analyze the following structured diagnostic evidence for target %q (%s protocol).

Evidence & Failure Graph:
%s

Provide a concise, expert Unix-engineer causal explanation. Format your answer with these exact markdown sections:
1. **Primary Root Cause**: A 1-2 sentence precise explanation of why the failure occurred at this exact stage.
2. **Contributing Factors**: Bullet points of hidden or secondary possibilities (e.g. firewall, DNS cache, MTU, file permissions, daemon config).
3. **Step-by-Step Remediation**: Copy-pasteable terminal commands to fix or verify the issue.
Keep your response professional, direct, and free of conversational fluff.`, diag.Target, diag.Protocol, string(jsonBytes))
}

// OllamaProvider supports offline local LLMs
type OllamaProvider struct{}

func (o *OllamaProvider) Name() string { return "Ollama (Local)" }

func (o *OllamaProvider) host() string {
	if h := os.Getenv("OLLAMA_HOST"); h != "" {
		return h
	}
	return "http://127.0.0.1:11434"
}

func (o *OllamaProvider) model() string {
	if m := os.Getenv("OLLAMA_MODEL"); m != "" {
		return m
	}
	return "llama3.2"
}

func (o *OllamaProvider) Available() bool {
	client := http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(o.host() + "/api/tags")
	if err == nil && resp.StatusCode == 200 {
		_ = resp.Body.Close()
		return true
	}
	return false
}

func (o *OllamaProvider) Analyze(ctx context.Context, diag *model.Diagnostic) (string, error) {
	prompt := buildPrompt(diag)
	reqBody := map[string]interface{}{
		"model":  o.model(),
		"prompt": prompt,
		"stream": false,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", o.host()+"/api/generate", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Response, nil
}

// GeminiProvider supports Google Gemini API
type GeminiProvider struct{}

func (g *GeminiProvider) Name() string { return "Google Gemini" }

func (g *GeminiProvider) Available() bool {
	return os.Getenv("GEMINI_API_KEY") != ""
}

func (g *GeminiProvider) Analyze(ctx context.Context, diag *model.Diagnostic) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
	prompt := buildPrompt(diag)

	reqPayload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	payloadBytes, _ := json.Marshal(reqPayload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("empty response from Gemini API")
}

// OpenAIProvider supports OpenAI and compatible APIs
type OpenAIProvider struct{}

func (o *OpenAIProvider) Name() string { return "OpenAI" }

func (o *OpenAIProvider) Available() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

func (o *OpenAIProvider) baseURL() string {
	if b := os.Getenv("OPENAI_BASE_URL"); b != "" {
		return strings.TrimSuffix(b, "/")
	}
	return "https://api.openai.com/v1"
}

func (o *OpenAIProvider) model() string {
	if m := os.Getenv("OPENAI_MODEL"); m != "" {
		return m
	}
	return "gpt-4o-mini"
}

func (o *OpenAIProvider) Analyze(ctx context.Context, diag *model.Diagnostic) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	prompt := buildPrompt(diag)

	reqBody := map[string]interface{}{
		"model": o.model(),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL()+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(b))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response from OpenAI API")
}

// AnthropicProvider supports Claude
type AnthropicProvider struct{}

func (a *AnthropicProvider) Name() string { return "Anthropic Claude" }

func (a *AnthropicProvider) Available() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

func (a *AnthropicProvider) Analyze(ctx context.Context, diag *model.Diagnostic) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	prompt := buildPrompt(diag)

	reqBody := map[string]interface{}{
		"model":      "claude-3-5-haiku-20241022",
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(b))
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) > 0 {
		return anthropicResp.Content[0].Text, nil
	}
	return "", fmt.Errorf("empty response from Anthropic API")
}
