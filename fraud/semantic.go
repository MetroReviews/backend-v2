package fraud

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const embeddingModel = "text-embedding-3-small"
const embeddingTimeout = 8 * time.Second

// Cosine similarity above which two reviews are treated as a reworded/paraphrased duplicate
// rather than merely on the same topic. text-embedding-3-small puts genuine paraphrases in the
// 0.9+ range and unrelated reviews well below that, but this is a heuristic, not a proof — tune
// it against real traffic before trusting it unsupervised.
const semanticDuplicateThreshold = 0.92

// The 30-day lookback mirrors IsDuplicate's same-author window, scoped instead to "same
// subject" so it also catches a spam campaign spreading one reworded review across many
// different accounts targeting the same business/project.

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed returns an OpenAI embedding for the given text, or nil if OpenAI isn't configured or
// the call fails. It fails open — a nil result just means "the semantic-duplicate check will be
// skipped for this review," never an error that blocks posting.
func Embed(ctx context.Context, text string) []float32 {
	apiKey := state.Config.OpenAI.APIKey
	text = strings.TrimSpace(text)
	if apiKey == "" || text == "" {
		return nil
	}

	body, err := json.Marshal(embeddingRequest{Model: embeddingModel, Input: text})
	if err != nil {
		state.Logger.Error("[fraud] failed to marshal embedding request", zap.Error(err))
		return nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, embeddingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		state.Logger.Error("[fraud] failed to build embedding request", zap.Error(err))
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		state.Logger.Warn("[fraud] embedding request failed; skipping semantic duplicate check", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		state.Logger.Warn("[fraud] embedding non-200 response; skipping semantic duplicate check", zap.Int("status", resp.StatusCode))
		return nil
	}

	var parsed embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Data) == 0 {
		state.Logger.Warn("[fraud] failed to decode embedding response; skipping semantic duplicate check", zap.Error(err))
		return nil
	}

	return parsed.Data[0].Embedding
}

// IsSemanticDuplicate reports whether embedding is a near-duplicate of another review already
// posted on the same subject (business or project) within the last 30 days — regardless of who
// wrote it. Pass a nil embedding (e.g. because Embed skipped/failed) to always get false.
func IsSemanticDuplicate(ctx context.Context, businessID, projectID *uuid.UUID, embedding []float32) (bool, error) {
	if len(embedding) == 0 {
		return false, nil
	}

	var query string
	var arg any
	switch {
	case businessID != nil:
		query = "SELECT embedding FROM reviews WHERE business_id = $1 AND embedding IS NOT NULL AND created_at >= NOW() - INTERVAL '30 days'"
		arg = *businessID
	case projectID != nil:
		query = "SELECT embedding FROM reviews WHERE project_id = $1 AND embedding IS NOT NULL AND created_at >= NOW() - INTERVAL '30 days'"
		arg = *projectID
	default:
		return false, nil
	}

	rows, err := state.Pool.Query(ctx, query, arg)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var candidate []float32
		if err := rows.Scan(&candidate); err != nil {
			return false, err
		}
		if cosineSimilarity(embedding, candidate) >= semanticDuplicateThreshold {
			return true, nil
		}
	}
	return false, rows.Err()
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
