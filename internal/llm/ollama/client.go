package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"golang.org/x/time/rate"
)

type Client struct {
	baseURL        string
	model          string
	http           *http.Client
	rateLimitCache *redis.Client
	llmRateLimiter *rate.Limiter
	configCache    *configs.ConfigCache
}

func NewClient(baseURL, model string, timeout time.Duration, rateLimitCache *redis.Client,
	configCache *configs.ConfigCache, llmRateLimiter *rate.Limiter) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		model:   model,
		http: &http.Client{
			Timeout: timeout,
		},
		rateLimitCache: rateLimitCache,
		configCache:    configCache,
		llmRateLimiter: llmRateLimiter,
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Think  bool   `json:"think"`
	System string `json:"system,omitempty"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

func (c *Client) Generate(ctx context.Context, userId int64, prompt string) (*GenerateResponse, error) {
	userDenied := false

	key := strconv.FormatInt(userId, 10)

	ok, setErr := c.rateLimitCache.SetNX(ctx, key, "", time.Minute).Result()
	if setErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot set rate limit: %s", setErr), "payload", key)
		ok = false
	}
	userDenied = !ok

	if userDenied {
		return nil, nil
	}

	if limiterErr := c.llmRateLimiter.Wait(ctx); limiterErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("LLM rate limiter error: %s", limiterErr))
	}

	reqBody := generateRequest{
		Model:  c.model,
		Think:  false,
		System: c.configCache.GetString("llm_system"),
		Prompt: prompt,
		Stream: false,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return &GenerateResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return &GenerateResponse{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return &GenerateResponse{}, fmt.Errorf("do request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("read closer err: %v", err))
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &GenerateResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &GenerateResponse{}, fmt.Errorf("ollama http %d: %s", resp.StatusCode, string(body))
	}

	var out GenerateResponse
	if err = json.Unmarshal(body, &out); err != nil {
		return &GenerateResponse{}, fmt.Errorf("unmarshal response: %w; body=%s", err, string(body))
	}

	if strings.TrimSpace(prompt) == "" {
		return &out, fmt.Errorf("empty prompt (done_reason=%s)", out.DoneReason)
	}

	return &out, nil
}
