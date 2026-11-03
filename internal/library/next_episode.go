package library

import (
	"context"
	"database/sql"
	"errors"
)

// nextEpisodeDTO es la parte del siguiente episodio que necesita el
// reproductor para encadenar la reproducción.
type nextEpisodeDTO struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
}

// queryNextEpisode busca el episodio inmediatamente posterior por temporada y
// número, saltando a la primera temporada siguiente que tenga episodios.
func queryNextEpisode(
	ctx context.Context,
	db *sql.DB,
	itemID int64,
	access libraryAccess,
) (*nextEpisodeDTO, error) {
	var next nextEpisodeDTO
	err := db.QueryRowContext(ctx, `
		WITH current AS (
			SELECT series_id, season, episode
			  FROM media_items
			 WHERE id = ? AND kind = 'episode'
			   AND season IS NOT NULL AND episode IS NOT NULL
		)
		SELECT n.id,
		       COALESCE(n.episode_title, n.display_title, n.title),
		       n.season, n.episode
		  FROM current c
		  JOIN media_items n
		    ON n.series_id = c.series_id
		   AND n.kind = 'episode'
		   AND n.season IS NOT NULL
		   AND n.episode IS NOT NULL
		   AND (n.season > c.season OR (n.season = c.season AND n.episode > c.episode))
		  JOIN series s ON s.id = n.series_id AND `+access.Clause("s")+`
		 WHERE `+access.Clause("n")+`
		 ORDER BY n.season, n.episode, n.id
		 LIMIT 1`, itemID).Scan(&next.ID, &next.Title, &next.Season, &next.Episode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &next, nil
}
