-- Colecciones de películas y sus miembros.
CREATE TABLE collections (
    id                 INTEGER PRIMARY KEY,
    name               TEXT    NOT NULL,
    overview           TEXT,
    poster_url         TEXT,
    backdrop_url       TEXT,
    tmdb_collection_id INTEGER UNIQUE,
    source             TEXT    NOT NULL CHECK (source IN ('tmdb', 'manual')),
    edited             INTEGER NOT NULL DEFAULT 0,
    deleted            INTEGER NOT NULL DEFAULT 0,
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

CREATE TABLE collection_items (
    collection_id INTEGER NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    media_item_id INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    position      INTEGER,
    excluded      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, media_item_id)
) STRICT;

CREATE INDEX collection_items_media_item_id ON collection_items (media_item_id);

ALTER TABLE media_items ADD COLUMN collection_checked INTEGER NOT NULL DEFAULT 0;
