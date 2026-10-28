-- Las sesiones guardan el dispositivo y el cliente que las abrió. Se recrea la
-- tabla para que created_at y last_seen_at tengan un defecto con función, algo
-- que ALTER TABLE no admite.
CREATE TABLE sessions_new (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   TEXT    NOT NULL UNIQUE,
    expires_at   TEXT    NOT NULL,
    device_name  TEXT    NOT NULL DEFAULT '',
    client       TEXT    NOT NULL DEFAULT 'web' CHECK (client IN ('web', 'android', 'androidtv')),
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    last_seen_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

INSERT INTO sessions_new (
    id, user_id, token_hash, expires_at, device_name, client, created_at, last_seen_at
)
SELECT id, user_id, token_hash, expires_at, '', 'web',
       strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
       strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
  FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX sessions_user_id ON sessions (user_id);
CREATE INDEX sessions_expires_at ON sessions (expires_at);
