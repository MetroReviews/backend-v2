-- Stores an OpenAI embedding for each review body so fraud.IsSemanticDuplicate can catch
-- reworded/paraphrased spam-campaign reviews across *different* authors on the same subject,
-- which the pg_trgm-based same-author check in fraud.IsDuplicate can't see. NULL whenever
-- openai.api_key isn't configured or the embeddings call fails — that check is skipped, not
-- an error, for any review with no embedding stored.
--
-- Stored as a plain array rather than a pgvector column so this doesn't require the pgvector
-- extension to be installed on the target Postgres instance; comparison happens in Go against
-- a small, subject-scoped candidate set rather than in SQL.
ALTER TABLE reviews ADD COLUMN embedding REAL[];

CREATE INDEX IF NOT EXISTS idx_reviews_business_embedding_created
	ON reviews (business_id, created_at) WHERE embedding IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_reviews_project_embedding_created
	ON reviews (project_id, created_at) WHERE embedding IS NOT NULL;
