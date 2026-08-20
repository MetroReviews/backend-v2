package fraud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"go.uber.org/zap"
)

const classifierModel = "gpt-4.1-mini"
const classifierTimeout = 10 * time.Second
const maxClassifierReason = 200

const classifierSystemPrompt = `You are a fraud-detection classifier for a business review platform. Given a star rating and review text, decide whether it looks like a fake, templated, incentivized, or bot-generated review rather than genuine customer feedback.

Genuine reviews are usually specific: they mention concrete details (a dish, a staff member, a specific problem or moment) and often have minor stylistic imperfections.

Treat a review as suspicious if it shows things like: generic praise/complaint with no specific details ("Great service, highly recommend!" and nothing else), overtly promotional or SEO-style language, a request for a reciprocal review or contact info/links embedded in the text, or a clear mismatch between the star rating and the sentiment of the written text (e.g. 5 stars paired with a complaint-shaped body).

Respond only with the requested JSON. Do not flag a review merely for being short, critical, or unusually positive — those alone are not evidence of fraud.`

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type jsonSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

type responseFormat struct {
	Type       string     `json:"type"`
	JSONSchema jsonSchema `json:"json_schema"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
	Temperature    float64        `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type authenticityResult struct {
	Suspicious bool   `json:"suspicious"`
	Reason     string `json:"reason"`
}

var authenticitySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"suspicious": map[string]any{"type": "boolean"},
		"reason":     map[string]any{"type": "string", "description": "One short sentence explaining the verdict"},
	},
	"required":             []string{"suspicious", "reason"},
	"additionalProperties": false,
}

// ClassifyAuthenticity asks an LLM whether a review reads as fake/templated/incentivized rather
// than genuine customer feedback. It's a no-op (never flags) if OpenAI isn't configured, and
// fails open — logging and returning unflagged — on any request/response error, exactly like
// the moderation package's Check, so an OpenAI outage never blocks a user from posting.
//
// This is the most expensive and slowest of the fraud checks (a full chat completion, versus a
// DB query or a single embedding call) — callers should run it last, after the cheaper checks
// have had a chance to short-circuit.
func ClassifyAuthenticity(ctx context.Context, rating int16, title, body string) (suspicious bool, reason string) {
	apiKey := state.Config.OpenAI.APIKey
	if apiKey == "" {
		return false, ""
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return false, ""
	}

	userContent := fmt.Sprintf("Rating: %d/5\nTitle: %s\nBody: %s", rating, title, body)

	reqBody, err := json.Marshal(chatRequest{
		Model: classifierModel,
		Messages: []chatMessage{
			{Role: "system", Content: classifierSystemPrompt},
			{Role: "user", Content: userContent},
		},
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchema{
				Name:   "review_authenticity",
				Strict: true,
				Schema: authenticitySchema,
			},
		},
		Temperature: 0,
	})
	if err != nil {
		state.Logger.Error("[fraud] failed to marshal classifier request", zap.Error(err))
		return false, ""
	}

	reqCtx, cancel := context.WithTimeout(ctx, classifierTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		state.Logger.Error("[fraud] failed to build classifier request", zap.Error(err))
		return false, ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		state.Logger.Warn("[fraud] classifier request failed; allowing content through", zap.Error(err))
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		state.Logger.Warn("[fraud] classifier non-200 response; allowing content through", zap.Int("status", resp.StatusCode))
		return false, ""
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Choices) == 0 {
		state.Logger.Warn("[fraud] failed to decode classifier response; allowing content through", zap.Error(err))
		return false, ""
	}

	var result authenticityResult
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &result); err != nil {
		state.Logger.Warn("[fraud] failed to parse classifier verdict; allowing content through", zap.Error(err))
		return false, ""
	}

	if !result.Suspicious {
		return false, ""
	}

	verdictReason := strings.TrimSpace(result.Reason)
	if len(verdictReason) > maxClassifierReason {
		verdictReason = verdictReason[:maxClassifierReason]
	}
	return true, "flagged by AI authenticity check: " + verdictReason
}
