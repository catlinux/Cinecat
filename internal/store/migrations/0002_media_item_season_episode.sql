-- Temporada y episodio de las entradas de series (Fase 1, paso 4).
ALTER TABLE media_items ADD COLUMN season INTEGER;
ALTER TABLE media_items ADD COLUMN episode INTEGER;

CREATE INDEX media_items_season_episode ON media_items (library_id, season, episode);
