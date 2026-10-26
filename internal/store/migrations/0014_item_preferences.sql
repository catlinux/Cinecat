-- Preferencias de reproducción de cada usuario para cada ítem. NULL significa
-- que no hay preferencia; subtitle_id vacío significa «sin subtítulos».
CREATE TABLE item_preferences (
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    media_item_id INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    audio_index   INTEGER,
    subtitle_id   TEXT,
    quality       TEXT,
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    PRIMARY KEY (user_id, media_item_id)
) STRICT;
