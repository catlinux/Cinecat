-- Ajuste por usuario para reproducir automáticamente el siguiente episodio.
-- Las series viven en la tabla series, no en media_items.
CREATE TABLE series_autoplay (
    user_id   INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    series_id INTEGER NOT NULL REFERENCES series (id) ON DELETE CASCADE,
    enabled   INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    PRIMARY KEY (user_id, series_id)
) STRICT;

CREATE INDEX series_autoplay_series_id ON series_autoplay (series_id);
