-- Backfill URL slugs for any business missing one (NULL or empty). Historically
-- the slug came straight from the client, so a submission without one was stored
-- unreachable (lookup is by slug). Slugs are now server-generated; this repairs
-- existing rows so every business has a working page.
DO $$
DECLARE
    b         RECORD;
    base      TEXT;
    candidate TEXT;
    n         INT;
BEGIN
    FOR b IN
        SELECT id, name FROM businesses
        WHERE slug IS NULL OR btrim(slug) = ''
    LOOP
        -- Mirror helpers.Slugify: lowercase, non-alphanumeric runs → '-', trim.
        base := btrim(regexp_replace(lower(coalesce(b.name, '')), '[^a-z0-9]+', '-', 'g'), '-');
        IF base = '' THEN
            base := 'business';
        END IF;

        candidate := base;
        n := 1;
        WHILE EXISTS (SELECT 1 FROM businesses WHERE slug = candidate) LOOP
            n := n + 1;
            candidate := base || '-' || n;
        END LOOP;

        UPDATE businesses SET slug = candidate, updated_at = NOW() WHERE id = b.id;
    END LOOP;
END $$;
