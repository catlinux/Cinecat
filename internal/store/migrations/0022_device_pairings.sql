-- Vinculación de dispositivos nativos mediante un código temporal. Solo se
-- guardan los hashes del secreto y del código; el secreto nunca se muestra.
CREATE TABLE device_pairings (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    pair_id          TEXT    NOT NULL UNIQUE,
    secret_hash      TEXT    NOT NULL,
    code_hash        TEXT    NOT NULL UNIQUE,
    device_name      TEXT    NOT NULL DEFAULT '',
    client           TEXT    NOT NULL CHECK (client IN ('android', 'androidtv')),
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at       TEXT    NOT NULL,
    last_polled_at   TEXT,
    approved_user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
    approved_at      TEXT,
    consumed_at      TEXT
) STRICT;

CREATE INDEX device_pairings_expires_at ON device_pairings (expires_at);
CREATE INDEX device_pairings_approved_user_id ON device_pairings (approved_user_id);

-- Si se borra el usuario aprobador, la vinculación se conserva sin dueño para
-- poder responder 410 al dispositivo y no entregar una sesión huérfana.
CREATE TRIGGER device_pairings_orphan_on_user_delete
BEFORE DELETE ON users
BEGIN
    UPDATE device_pairings
       SET approved_user_id = NULL
     WHERE approved_user_id = OLD.id;
END;
