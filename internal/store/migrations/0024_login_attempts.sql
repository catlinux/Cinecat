-- Los intentos fallidos y sus bloqueos dejan de vivir en memoria: así
-- persisten entre reinicios y se comparten entre instancias.
CREATE TABLE login_attempts (
    key          TEXT    NOT NULL,
    kind         TEXT    NOT NULL CHECK (kind IN ('user', 'ip')),
    count        INTEGER NOT NULL DEFAULT 0,
    last_at      TEXT    NOT NULL,
    locked_until TEXT,
    PRIMARY KEY (kind, key)
) STRICT;

CREATE INDEX login_attempts_last_at ON login_attempts (last_at);

CREATE TABLE login_lockouts (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    kind         TEXT    NOT NULL CHECK (kind IN ('user', 'ip')),
    username     TEXT,
    ip           TEXT,
    locked_at    TEXT    NOT NULL,
    locked_until TEXT    NOT NULL
) STRICT;

CREATE INDEX login_lockouts_locked_at ON login_lockouts (locked_at);
