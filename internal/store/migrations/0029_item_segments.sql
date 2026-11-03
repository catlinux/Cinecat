-- Segmentos manuales de intro y créditos; los detectados en los capítulos no se guardan.
CREATE TABLE item_segments (
    item_id       INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    kind          TEXT    NOT NULL CHECK (kind IN ('intro', 'credits')),
    start_seconds REAL    NOT NULL,
    end_seconds   REAL    NOT NULL,
    PRIMARY KEY (item_id, kind)
) STRICT;
