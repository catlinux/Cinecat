-- Series, temporadas y episodios (plan fichas-tmdb, paso 11).
CREATE TABLE series (
    id               INTEGER PRIMARY KEY,
    library_id       INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    folder           TEXT    NOT NULL COLLATE NOCASE,
    title            TEXT    NOT NULL,
    display_title    TEXT,
    original_title   TEXT,
    year             INTEGER,
    tagline          TEXT,
    overview         TEXT,
    overview_language TEXT,
    poster_url       TEXT,
    backdrop_url     TEXT,
    genres           TEXT,
    rating           REAL,
    directors        TEXT,
    cast_names       TEXT,
    tmdb_id          INTEGER,
    metadata_source  TEXT    CHECK (metadata_source IN ('tmdb')),
    match_status     TEXT    CHECK (match_status IN ('auto', 'manual', 'none')),
    match_score      REAL,
    sort_title       TEXT,
    sort_letter      TEXT,
    poster_locked    INTEGER NOT NULL DEFAULT 0,
    backdrop_locked  INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (library_id, folder)
) STRICT;

CREATE TABLE series_seasons (
    series_id INTEGER NOT NULL REFERENCES series (id) ON DELETE CASCADE,
    season    INTEGER NOT NULL,
    name      TEXT,
    overview  TEXT,
    poster_url TEXT,
    PRIMARY KEY (series_id, season)
) STRICT;

ALTER TABLE media_items ADD COLUMN series_id INTEGER REFERENCES series (id) ON DELETE SET NULL;
ALTER TABLE media_items ADD COLUMN episode_title TEXT;
ALTER TABLE media_items ADD COLUMN still_url TEXT;

CREATE INDEX media_items_series_id ON media_items (series_id);
