-- Subtítulos externos subidos por los administradores. Los archivos viven
-- junto a la base de datos, fuera de las carpetas de vídeo.
CREATE TABLE item_subtitles (
    id            INTEGER PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    language      TEXT    NOT NULL DEFAULT '',
    label         TEXT    NOT NULL DEFAULT '',
    filename      TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

CREATE INDEX item_subtitles_item ON item_subtitles (media_item_id);
