-- Ficha completa de TMDB para películas y estado de la identificación.
ALTER TABLE media_items ADD COLUMN original_title TEXT;
ALTER TABLE media_items ADD COLUMN imdb_id TEXT;
ALTER TABLE media_items ADD COLUMN tagline TEXT;
ALTER TABLE media_items ADD COLUMN genres TEXT;
ALTER TABLE media_items ADD COLUMN runtime_minutes INTEGER;
ALTER TABLE media_items ADD COLUMN rating REAL;
ALTER TABLE media_items ADD COLUMN directors TEXT;
ALTER TABLE media_items ADD COLUMN cast_names TEXT;
ALTER TABLE media_items ADD COLUMN backdrop_url TEXT;
ALTER TABLE media_items ADD COLUMN overview_language TEXT;
ALTER TABLE media_items ADD COLUMN metadata_source TEXT CHECK (metadata_source IN ('tmdb', 'omdb'));
ALTER TABLE media_items ADD COLUMN match_status TEXT CHECK (match_status IN ('auto', 'manual', 'none'));
ALTER TABLE media_items ADD COLUMN match_score REAL;

-- Los metadatos que ya había venían de OMDb.
UPDATE media_items
SET metadata_source = 'omdb'
WHERE overview IS NOT NULL OR poster_url IS NOT NULL;
