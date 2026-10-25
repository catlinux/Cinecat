-- Esquema inicial de Cinecat (Fase 1, paso 2).
-- Las fechas son texto ISO 8601 en UTC. Las claves foráneas se activan por
-- conexión desde la cadena de conexión (ver internal/store/store.go).

CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

CREATE TABLE sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    expires_at TEXT    NOT NULL
) STRICT;

CREATE INDEX sessions_user_id ON sessions (user_id);
CREATE INDEX sessions_expires_at ON sessions (expires_at);

CREATE TABLE libraries (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT    NOT NULL UNIQUE,
    name TEXT    NOT NULL
) STRICT;

CREATE TABLE media_items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    path       TEXT    NOT NULL,
    title      TEXT    NOT NULL,
    year       INTEGER,
    kind       TEXT    NOT NULL CHECK (kind IN ('movie', 'series', 'episode')),
    tmdb_id    INTEGER,
    UNIQUE (library_id, path)
) STRICT;

CREATE INDEX media_items_library_id ON media_items (library_id);
CREATE INDEX media_items_tmdb_id ON media_items (tmdb_id);

CREATE TABLE playback_state (
    user_id          INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    media_item_id    INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    position_seconds INTEGER NOT NULL DEFAULT 0 CHECK (position_seconds >= 0),
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (user_id, media_item_id)
) STRICT;

CREATE INDEX playback_state_media_item_id ON playback_state (media_item_id);
