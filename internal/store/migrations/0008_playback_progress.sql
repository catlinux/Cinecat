-- Progreso de reproducción por usuario e ítem.
CREATE TABLE playback_progress (
    user_id          INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    media_item_id    INTEGER NOT NULL REFERENCES media_items (id) ON DELETE CASCADE,
    position_seconds REAL    NOT NULL,
    duration_seconds REAL    NOT NULL,
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (user_id, media_item_id)
) STRICT;

CREATE INDEX playback_progress_user_updated
    ON playback_progress (user_id, updated_at DESC);
