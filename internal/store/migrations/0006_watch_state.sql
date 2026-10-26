-- Clave normalizada para ordenar y filtrar por la inicial del título.
ALTER TABLE media_items ADD COLUMN sort_title TEXT;
ALTER TABLE media_items ADD COLUMN sort_letter TEXT;

-- Estado de visionado por usuario.
CREATE TABLE watch_state (
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    media_item_id INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    watched_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (user_id, media_item_id)
) STRICT;

CREATE INDEX watch_state_media_item_id ON watch_state (media_item_id);
