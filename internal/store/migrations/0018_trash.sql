-- Papelera de la biblioteca de vídeo. La fila guarda lo necesario para
-- devolver el archivo y su ficha sin conservar entradas en media_items.
CREATE TABLE trash_items (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id    INTEGER NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    original_path TEXT    NOT NULL,
    trash_path    TEXT    NOT NULL,
    moved_kind    TEXT    NOT NULL CHECK (moved_kind IN ('file', 'folder')),
    extra_paths   TEXT    NOT NULL DEFAULT '[]',
    title         TEXT    NOT NULL,
    year          INTEGER,
    kind          TEXT    NOT NULL,
    snapshot      TEXT    NOT NULL,
    deleted_by    TEXT,
    deleted_at    TEXT    NOT NULL
) STRICT;

CREATE INDEX trash_items_deleted_at ON trash_items (deleted_at);
