package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds AI service configuration.
type Config struct {
	Provider string // "nim", "groq", "openai", "ollama"
	APIKey   string
	Model    string
}

// Client handles LLM communications.
type Client struct {
	config Config
	http   *http.Client
}

// NewClient initializes an AI client.
func NewClient(cfg Config) *Client {
	loadDotEnv()

	if cfg.Provider == "" {
		cfg.Provider = "nim"
	}
	if cfg.APIKey == "" {
		cfg.APIKey = getAPIKeyFromEnv(cfg.Provider)
	}
	if cfg.Model == "" {
		switch cfg.Provider {
		case "nim":
			cfg.Model = "meta/llama-3.3-70b-instruct"
		case "groq":
			cfg.Model = "llama-3.3-70b-versatile"
		case "openai":
			cfg.Model = "gpt-4o-mini"
		case "ollama":
			cfg.Model = "llama3"
		}
	}
	return &Client{
		config: cfg,
		http:   &http.Client{Timeout: 120 * time.Second},
	}
}

func getAPIKeyFromEnv(provider string) string {
	switch provider {
	case "nim":
		key := os.Getenv("NVIDIA_NIM_API_KEY")
		if key == "" {
			key = os.Getenv("NVIDIA_API_KEY")
		}
		return key
	case "groq":
		keys := os.Getenv("GROQ_API_KEYS")
		if keys == "" {
			keys = os.Getenv("GROQ_API_KEY")
		}
		if keys != "" {
			parts := strings.Split(keys, ",")
			return strings.TrimSpace(parts[0])
		}
		return ""
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	}
	return ""
}

func loadDotEnv() {
	candidatePaths := []string{
		".env",
		"../.env",
		"../../.env",
		"/Users/adityakinjawadekar/Documents/100xcode/open-pptx/.env",
	}

	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidatePaths = append(candidatePaths,
			filepath.Join(exeDir, ".env"),
			filepath.Join(exeDir, "../Resources/.env"),
			filepath.Join(exeDir, "../../../.env"),
			filepath.Join(exeDir, "../../../../.env"),
		)
	}

	for _, p := range candidatePaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if os.Getenv(k) == "" {
					os.Setenv(k, v)
				}
			}
		}
	}
}

// ChatMessage represents a single chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the OpenAI-compatible request body.
type ChatRequest struct {
	Model          string         `json:"model"`
	Messages       []ChatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat controls structured output.
type ResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

// ChatResponse is the OpenAI-compatible response.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a system+user prompt to the LLM and returns the response text.
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	apiKey := c.config.APIKey
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(c.config.Provider)
	}
	if apiKey == "" && c.config.Provider != "ollama" {
		return "", fmt.Errorf("API key for provider '%s' is missing", c.config.Provider)
	}

	endpoint := c.getEndpoint()

	reqBody := ChatRequest{
		Model: c.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	}

	if c.config.Provider == "nim" || c.config.Provider == "groq" {
		reqBody.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request to %s: %w", c.config.Provider, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s API returned status %d: %s", c.config.Provider, resp.StatusCode, string(respBytes))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w (raw: %s)", err, string(respBytes))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("%s API error: %s", c.config.Provider, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from %s API", c.config.Provider)
	}

	return chatResp.Choices[0].Message.Content, nil
}

// getEndpoint returns the API endpoint for the configured provider.
func (c *Client) getEndpoint() string {
	switch c.config.Provider {
	case "nim":
		return "https://integrate.api.nvidia.com/v1/chat/completions"
	case "groq":
		return "https://api.groq.com/openai/v1/chat/completions"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "ollama":
		return "http://localhost:11434/v1/chat/completions"
	default:
		return "https://integrate.api.nvidia.com/v1/chat/completions"
	}
}

// StreamChunk holds delta text and reasoning content delta.
type StreamChunk struct {
	Reasoning string `json:"reasoning"`
	Content   string `json:"content"`
}

type StreamEventChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
		} `json:"delta"`
	} `json:"choices"`
}

// CompleteStream streams reasoning and content via callback function in real-time.
func (c *Client) CompleteStream(ctx context.Context, systemPrompt, userPrompt string, callback func(chunk StreamChunk)) (string, error) {
	apiKey := c.config.APIKey
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(c.config.Provider)
	}
	if apiKey == "" && c.config.Provider != "ollama" {
		return "", fmt.Errorf("API key for provider '%s' is missing", c.config.Provider)
	}

	endpoint := c.getEndpoint()

	reqMap := map[string]interface{}{
		"model": c.config.Model,
		"messages": []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":   4096,
		"stream":       true,
	}

	bodyBytes, err := json.Marshal(reqMap)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request to %s: %w", c.config.Provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%s API returned status %d: %s", c.config.Provider, resp.StatusCode, string(respBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" {
				break
			}

			var eventChunk StreamEventChunk
			if err := json.Unmarshal([]byte(dataStr), &eventChunk); err == nil && len(eventChunk.Choices) > 0 {
				delta := eventChunk.Choices[0].Delta
				reasoning := delta.ReasoningContent
				if reasoning == "" {
					reasoning = delta.Reasoning
				}

				if delta.Content != "" {
					fullContent.WriteString(delta.Content)
				}

				if callback != nil && (delta.Content != "" || reasoning != "") {
					callback(StreamChunk{
						Reasoning: reasoning,
						Content:   delta.Content,
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullContent.String(), fmt.Errorf("error reading stream: %w", err)
	}

	return fullContent.String(), nil
}
