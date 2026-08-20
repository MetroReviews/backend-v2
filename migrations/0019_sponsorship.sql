
ALTER TABLE businesses ADD COLUMN featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE businesses ADD COLUMN featured_until TIMESTAMPTZ;
CREATE INDEX idx_businesses_featured ON businesses (featured) WHERE featured;
