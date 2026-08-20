
ALTER TABLE businesses ADD COLUMN latitude  DOUBLE PRECISION;
ALTER TABLE businesses ADD COLUMN longitude DOUBLE PRECISION;

ALTER TABLE businesses ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_businesses_search_vector ON businesses USING GIN (search_vector);

ALTER TABLE projects ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_projects_search_vector ON projects USING GIN (search_vector);
