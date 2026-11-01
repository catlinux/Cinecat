-- TMDB exige refrescar sus metadatos al menos cada seis meses. La fecha de
-- las fichas existentes empieza a contar ahora; las nuevas la rellenan al
-- guardar la ficha.
ALTER TABLE media_items ADD COLUMN metadata_refreshed_at TEXT;
ALTER TABLE series ADD COLUMN metadata_refreshed_at TEXT;

UPDATE media_items
   SET metadata_refreshed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
 WHERE metadata_source = 'tmdb';

UPDATE series
   SET metadata_refreshed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
 WHERE metadata_source = 'tmdb';
