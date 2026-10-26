-- Las cachés antiguas no guardaban todos los campos de TMDB. Se fuerza a que
-- el siguiente escaneo vuelva a pedir las fichas para reconstruir créditos,
-- fotos y colecciones.
UPDATE media_items SET credits = NULL, collection_checked = 0 WHERE metadata_source = 'tmdb';
UPDATE series SET credits = NULL WHERE metadata_source = 'tmdb';
