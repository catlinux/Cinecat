CREATE TABLE item_lists (
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    media_item_id INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    list          TEXT NOT NULL CHECK (list IN ('favorite', 'watchlist')),
    added_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, media_item_id, list)
) STRICT;

CREATE TABLE series_lists (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    series_id  INTEGER NOT NULL REFERENCES series (id) ON DELETE CASCADE,
    list       TEXT NOT NULL CHECK (list IN ('favorite', 'watchlist')),
    added_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, series_id, list)
) STRICT;

CREATE TABLE collection_lists (
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    collection_id INTEGER NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    list          TEXT NOT NULL CHECK (list IN ('favorite', 'watchlist')),
    added_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, collection_id, list)
) STRICT;

CREATE INDEX item_lists_user_list_added ON item_lists (user_id, list, added_at DESC);
CREATE INDEX series_lists_user_list_added ON series_lists (user_id, list, added_at DESC);
CREATE INDEX collection_lists_user_list_added ON collection_lists (user_id, list, added_at DESC);
