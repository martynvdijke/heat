package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"heat/ent/aisetting"
	"heat/models"
)

const (
	aiTextDefaultModel = "gpt-4o-mini"
	aiTextSystemPrompt = "You are a motorsport commentator writing short, punchy, original race commentary quotes. " +
		"Respond with ONLY a JSON array (no markdown fences, no prose) of objects with string keys \"text\" and \"author\". " +
		"Each quote must be a single sentence under 160 characters."
)

type aiTextRequest struct {
	Model       string      `json:"model"`
	Messages    []aiTextMsg `json:"messages"`
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

type aiTextMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiTextResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// QuoteSuggestRequest is the optional body for the quote suggestion endpoint.
type QuoteSuggestRequest struct {
	Context string `json:"context"`
	Count   int    `json:"count"`
}

// HandleQuoteSuggest asks the configured OpenAI-compatible text endpoint for
// quote suggestions. It follows the same provider-identification convention as
// track extraction: a heat/<version> User-Agent and a stable
// x-opencode-session that is echoed back to the caller.
//
// @Summary Suggest quotes with AI
// @Description Generate commentary quote suggestions using the configured AI text endpoint
// @Tags Quotes
// @Accept json
// @Produce json
// @Param request body QuoteSuggestRequest false "Optional context and count"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/quotes/ai-suggest [post]
func (h *Handler) HandleQuoteSuggest(c *gin.Context) {
	var input QuoteSuggestRequest
	// Body is optional; bind errors (e.g. empty body) are not fatal.
	_ = c.ShouldBindJSON(&input)
	if input.Count <= 0 || input.Count > 10 {
		input.Count = 3
	}

	baseURL := os.Getenv("AI_TEXT_GEN_URL")
	model := ""
	apiKey := ""
	setting, sErr := h.S.Ent.AISetting.Query().Where(aisetting.ID(1)).First(c.Request.Context())
	if sErr == nil {
		model = setting.TextGenModel
		apiKey = setting.APIKey
		if baseURL == "" && setting.Enabled == 1 {
			baseURL = setting.TextGenURL
		}
	}
	if strings.TrimSpace(model) == "" {
		model = os.Getenv("AI_TEXT_GEN_MODEL")
	}
	if strings.TrimSpace(model) == "" {
		model = aiTextDefaultModel
	}

	endpoint := aiRequestURL(baseURL, "/chat/completions")
	if endpoint == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "AI text endpoint not configured"})
		return
	}

	userPrompt := fmt.Sprintf("Generate %d original race commentary quotes.", input.Count)
	if ctx := strings.TrimSpace(input.Context); ctx != "" {
		userPrompt = fmt.Sprintf("Generate %d original race commentary quotes about: %s", input.Count, ctx)
	}

	payload := aiTextRequest{
		Model: model,
		Messages: []aiTextMsg{
			{Role: "system", Content: aiTextSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.9,
		MaxTokens:   900,
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to build AI request"})
		return
	}

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   60 * time.Second,
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create AI request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	sessionID := resolveSessionID(c.GetHeader("x-opencode-session"))
	setAIProviderHeaders(req, sessionID, h.S.CurrentVersion)
	c.Header("X-Opencode-Session", sessionID)

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "AI request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read AI response"})
		return
	}
	if resp.StatusCode >= 400 {
		h.S.Log.Errorf("ai", "HandleQuoteSuggest: provider status %d", resp.StatusCode)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "AI request failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": parseQuoteSuggestions(bodyBytes),
		"session_id":  sessionID,
		"model":       model,
	})
}

// parseQuoteSuggestions extracts quote candidates from an OpenAI-compatible
// chat completion response. It accepts a JSON array (possibly wrapped in
// markdown fences) and falls back to line-splitting plain text.
func parseQuoteSuggestions(body []byte) []models.QuoteSuggestion {
	var completion aiTextResponse
	if err := json.Unmarshal(body, &completion); err != nil || len(completion.Choices) == 0 {
		return []models.QuoteSuggestion{}
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var suggestions []models.QuoteSuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err == nil {
		out := suggestions[:0]
		for _, s := range suggestions {
			if strings.TrimSpace(s.Text) == "" {
				continue
			}
			if strings.TrimSpace(s.Author) == "" {
				s.Author = "Commentator"
			}
			out = append(out, s)
		}
		if len(out) > 0 {
			return out
		}
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.Trim(line, "\"-•"))
		if line == "" {
			continue
		}
		suggestions = append(suggestions, models.QuoteSuggestion{Text: line, Author: "Commentator"})
	}
	if suggestions == nil {
		return []models.QuoteSuggestion{}
	}
	return suggestions
}
