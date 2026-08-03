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
	"sync"
	"time"
)

var (
	groqMu     sync.Mutex
	groqKeyIdx int
)

// Config holds AI service configuration.
type Config struct {
	Provider string // "groq", "nim", "openai", "ollama"
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
		cfg.Provider = "groq"
	}
	if cfg.Model == "" {
		switch cfg.Provider {
		case "groq":
			cfg.Model = "llama-3.3-70b-versatile"
		case "nim":
			cfg.Model = "meta/llama-3.3-70b-instruct"
		case "openai":
			cfg.Model = "gpt-4o-mini"
		case "ollama":
			cfg.Model = "llama3"
		}
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
	}

	return &Client{
		config: cfg,
		http: &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		},
	}
}

// Get all Groq keys in rotation
func getGroqKeyList() []string {
	loadDotEnv()
	raw := os.Getenv("GROQ_API_KEYS")
	if raw == "" {
		raw = os.Getenv("GROQ_API_KEY")
	}
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	var valid []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			valid = append(valid, trimmed)
		}
	}
	return valid
}

func getNextGroqKey() string {
	keys := getGroqKeyList()
	if len(keys) == 0 {
		return ""
	}
	groqMu.Lock()
	defer groqMu.Unlock()
	key := keys[groqKeyIdx%len(keys)]
	groqKeyIdx++
	return key
}

func getAPIKeyFromEnv(provider string) string {
	switch provider {
	case "groq":
		return getNextGroqKey()
	case "nim":
		key := os.Getenv("NVIDIA_NIM_API_KEY")
		if key == "" {
			key = os.Getenv("NVIDIA_API_KEY")
		}
		return key
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

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string         `json:"model"`
	Messages       []ChatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

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

// CompleteStream streams reasoning and content with key rotation retries.
func (c *Client) CompleteStream(ctx context.Context, systemPrompt, userPrompt string, callback func(chunk StreamChunk)) (string, error) {
	if c.config.Provider == "groq" {
		keys := getGroqKeyList()
		if len(keys) == 0 {
			return "", fmt.Errorf("no Groq API keys found in .env")
		}
		var lastErr error
		// Rotate through keys on failure/rate-limit
		for i := 0; i < len(keys); i++ {
			apiKey := getNextGroqKey()
			res, err := c.completeStreamSingle(ctx, "groq", c.config.Model, apiKey, systemPrompt, userPrompt, callback)
			if err == nil {
				return res, nil
			}
			lastErr = err
			if callback != nil {
				callback(StreamChunk{Reasoning: fmt.Sprintf("\n[Key rotation: key %d rate-limited/failed, rotating to next key...]\n", i+1)})
			}
		}
		return "", fmt.Errorf("all Groq API keys failed: %w", lastErr)
	}

	apiKey := c.config.APIKey
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(c.config.Provider)
	}
	return c.completeStreamSingle(ctx, c.config.Provider, c.config.Model, apiKey, systemPrompt, userPrompt, callback)
}

func (c *Client) completeStreamSingle(ctx context.Context, provider, model, apiKey, systemPrompt, userPrompt string, callback func(chunk StreamChunk)) (string, error) {
	if apiKey == "" && provider != "ollama" {
		return "", fmt.Errorf("API key for provider '%s' is missing", provider)
	}

	endpoint := c.getEndpoint(provider)

	reqMap := map[string]interface{}{
		"model": model,
		"messages": []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":   8192,
		"stream":       true,
	}

	if provider == "nim" || provider == "groq" {
		reqMap["response_format"] = ResponseFormat{Type: "json_object"}
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
		return "", fmt.Errorf("http request to %s: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%s API returned status %d: %s", provider, resp.StatusCode, string(respBytes))
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

func (c *Client) getEndpoint(provider string) string {
	switch provider {
	case "groq":
		return "https://api.groq.com/openai/v1/chat/completions"
	case "nim":
		return "https://integrate.api.nvidia.com/v1/chat/completions"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "ollama":
		return "http://localhost:11434/v1/chat/completions"
	default:
		return "https://api.groq.com/openai/v1/chat/completions"
	}
}
