// Package moderation screens user-submitted text with OpenAI's Moderation API before it's
// published, catching hateful/harassing/violent/sexual content that the fraud package's
// heuristics (duplicate detection, account age) aren't meant to detect.
package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"go.uber.org/zap"
)

const apiURL = "https://api.openai.com/v1/moderations"
const model = "omni-moderation-latest"
const requestTimeout = 8 * time.Second

type moderationRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type moderationResult struct {
	Flagged    bool            `json:"flagged"`
	Categories map[string]bool `json:"categories"`
}

type moderationResponse struct {
	Results []moderationResult `json:"results"`
}

// Check runs the given text segments (e.g. a review's title and body) through OpenAI's
// Moderation API and reports whether they should be flagged, plus a human-readable reason
// naming the triggered categories.
//
// It is a no-op — never flags — if openai.api_key isn't configured, and fails open (logs and
// returns unflagged) on any request/response error, so an OpenAI outage or misconfiguration
// never blocks a user from posting.
func Check(ctx context.Context, texts ...string) (flagged bool, reason string) {
	apiKey := state.Config.OpenAI.APIKey
	if apiKey == "" {
		return false, ""
	}

	input := strings.TrimSpace(strings.Join(texts, "\n\n"))
	if input == "" {
		return false, ""
	}

	body, err := json.Marshal(moderationRequest{Model: model, Input: input})
	if err != nil {
		state.Logger.Error("[moderation] failed to marshal request", zap.Error(err))
		return false, ""
	}

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		state.Logger.Error("[moderation] failed to build request", zap.Error(err))
		return false, ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		state.Logger.Warn("[moderation] request failed; allowing content through", zap.Error(err))
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		state.Logger.Warn("[moderation] non-200 response; allowing content through", zap.Int("status", resp.StatusCode))
		return false, ""
	}

	var parsed moderationResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Results) == 0 {
		state.Logger.Warn("[moderation] failed to decode response; allowing content through", zap.Error(err))
		return false, ""
	}

	result := parsed.Results[0]
	if !result.Flagged {
		return false, ""
	}

	var triggered []string
	for category, hit := range result.Categories {
		if hit {
			triggered = append(triggered, category)
		}
	}
	sort.Strings(triggered)

	return true, "flagged by content moderation: " + strings.Join(triggered, ", ")
}
