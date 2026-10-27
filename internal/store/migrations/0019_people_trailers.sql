-- Enlaces de tráiler de TMDB para películas y series.
ALTER TABLE media_items ADD COLUMN trailer_url TEXT;
ALTER TABLE media_items ADD COLUMN trailer_language TEXT;

ALTER TABLE series ADD COLUMN trailer_url TEXT;
ALTER TABLE series ADD COLUMN trailer_language TEXT;

-- Los créditos antiguos no llevaban el id de la persona ni se guardó el
-- tráiler. Se vacían para que el siguiente escaneo vuelva a pedir la ficha.
UPDATE media_items SET credits = NULL WHERE metadata_source = 'tmdb';
UPDATE series SET credits = NULL WHERE metadata_source = 'tmdb';
