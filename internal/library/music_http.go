package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

type musicArtistDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AlbumCount int64  `json:"album_count"`
	TrackCount int64  `json:"track_count"`
}

type musicArtistsResponse struct {
	Artists []musicArtistDTO `json:"artists"`
	Total   int64            `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

type musicArtistDetailDTO struct {
	musicArtistDTO
	Albums []musicAlbumDTO `json:"albums"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type musicAlbumDTO struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Artist          string  `json:"artist"`
	ArtistID        int64   `json:"artist_id"`
	Year            *int    `json:"year"`
	TrackCount      int64   `json:"track_count"`
	DurationSeconds float64 `json:"duration_seconds"`
	HasCover        bool    `json:"has_cover"`
}

type musicAlbumsResponse struct {
	Albums []musicAlbumDTO `json:"albums"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type musicTrackDTO struct {
	ID              int64    `json:"id"`
	Title           string   `json:"title"`
	TrackNo         *int     `json:"track_no"`
	DiscNo          int      `json:"disc_no"`
	Artist          string   `json:"artist"`
	DurationSeconds *float64 `json:"duration_seconds"`
	Codec           *string  `json:"codec"`
	BitRate         *int64   `json:"bitrate"`
	Playable        bool     `json:"playable"`
}

type musicAlbumDetailDTO struct {
	musicAlbumDTO
	Tracks []musicTrackDTO `json:"tracks"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type musicTracksResponse struct {
	Tracks []musicTrackDTO `json:"tracks"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type musicPlayResponse struct {
	URL       string    `json:"url"`
	MIME      string    `json:"mime"`
	ExpiresAt time.Time `json:"expires_at"`
	Playable  bool      `json:"playable"`
}

func (a *API) handleMusicArtists(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	where, args, err := musicArtistWhere(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM music_artists ar WHERE `+where, args...).Scan(&total); err != nil {
		log.Printf("library: contar artistas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT ar.id, ar.name,
		       (SELECT count(*) FROM music_albums ma WHERE ma.artist_id = ar.id),
		       (SELECT count(*) FROM music_tracks mt WHERE mt.artist_id = ar.id)
		  FROM music_artists ar
		 WHERE `+where+`
		 ORDER BY ar.sort_name COLLATE NOCASE, ar.id
		 LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		log.Printf("library: listar artistas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()
	artists := make([]musicArtistDTO, 0)
	for rows.Next() {
		var artist musicArtistDTO
		if err := rows.Scan(&artist.ID, &artist.Name, &artist.AlbumCount, &artist.TrackCount); err != nil {
			log.Printf("library: leer artista: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		artists = append(artists, artist)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer artistas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, musicArtistsResponse{
		Artists: artists, Total: total, Limit: limit, Offset: offset,
	})
}

func musicArtistWhere(r *http.Request) (string, []any, error) {
	where := "1 = 1"
	args := make([]any, 0)
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		where += ` AND LOWER(ar.name) LIKE LOWER(?) ESCAPE '\'`
		args = append(args, likePattern(q))
	}
	letter := strings.TrimSpace(r.URL.Query().Get("letter"))
	if letter != "" {
		parsed, err := parseLetter(letter)
		if err != nil {
			return "", nil, err
		}
		condition, value := musicLetterCondition("ar.sort_name", parsed)
		where += " AND " + condition
		if value != nil {
			args = append(args, value)
		}
	}
	return where, args, nil
}

// musicLetterExpression acerca las iniciales acentuadas a su letra base para
// que los filtros A-Z funcionen también con nombres como «Óscar» o «Ñu».
func musicLetterExpression(column string) string {
	return `(CASE SUBSTR(` + column + `, 1, 1)
		WHEN 'Á' THEN 'A' WHEN 'À' THEN 'A' WHEN 'Ä' THEN 'A' WHEN 'Â' THEN 'A'
		WHEN 'É' THEN 'E' WHEN 'È' THEN 'E' WHEN 'Ë' THEN 'E' WHEN 'Ê' THEN 'E'
		WHEN 'Í' THEN 'I' WHEN 'Ì' THEN 'I' WHEN 'Ï' THEN 'I' WHEN 'Î' THEN 'I'
		WHEN 'Ó' THEN 'O' WHEN 'Ò' THEN 'O' WHEN 'Ö' THEN 'O' WHEN 'Ô' THEN 'O'
		WHEN 'Ú' THEN 'U' WHEN 'Ù' THEN 'U' WHEN 'Ü' THEN 'U' WHEN 'Û' THEN 'U'
		WHEN 'Ñ' THEN 'N' WHEN 'Ç' THEN 'C'
		WHEN 'á' THEN 'A' WHEN 'à' THEN 'A' WHEN 'ä' THEN 'A' WHEN 'â' THEN 'A'
		WHEN 'é' THEN 'E' WHEN 'è' THEN 'E' WHEN 'ë' THEN 'E' WHEN 'ê' THEN 'E'
		WHEN 'í' THEN 'I' WHEN 'ì' THEN 'I' WHEN 'ï' THEN 'I' WHEN 'î' THEN 'I'
		WHEN 'ó' THEN 'O' WHEN 'ò' THEN 'O' WHEN 'ö' THEN 'O' WHEN 'ô' THEN 'O'
		WHEN 'ú' THEN 'U' WHEN 'ù' THEN 'U' WHEN 'ü' THEN 'U' WHEN 'û' THEN 'U'
		WHEN 'ñ' THEN 'N' WHEN 'ç' THEN 'C'
		ELSE UPPER(SUBSTR(` + column + `, 1, 1)) END)`
}

func musicLetterCondition(column, letter string) (string, any) {
	expression := musicLetterExpression(column)
	if letter == "#" {
		return expression + " NOT BETWEEN 'A' AND 'Z'", nil
	}
	return expression + " = ?", letter
}

func (a *API) handleMusicArtist(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "artista no encontrado")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	artist, err := loadMusicArtist(r.Context(), a.db, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "artista no encontrado")
		return
	case err != nil:
		log.Printf("library: leer artista %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	albums, total, err := queryMusicAlbums(r.Context(), a.db, "a.artist_id = ?", []any{id},
		" ORDER BY a.year IS NULL, a.year, a.title COLLATE NOCASE", limit, offset)
	if err != nil {
		log.Printf("library: listar álbumes del artista %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, musicArtistDetailDTO{
		musicArtistDTO: artist, Albums: albums, Total: total, Limit: limit, Offset: offset,
	})
}

func loadMusicArtist(ctx context.Context, db *sql.DB, id int64) (musicArtistDTO, error) {
	var artist musicArtistDTO
	err := db.QueryRowContext(ctx, `
		SELECT ar.id, ar.name,
		       (SELECT count(*) FROM music_albums ma WHERE ma.artist_id = ar.id),
		       (SELECT count(*) FROM music_tracks mt WHERE mt.artist_id = ar.id)
		  FROM music_artists ar WHERE ar.id = ?`, id).
		Scan(&artist.ID, &artist.Name, &artist.AlbumCount, &artist.TrackCount)
	return artist, err
}

func (a *API) handleMusicAlbums(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	where := "1 = 1"
	args := make([]any, 0)
	query := r.URL.Query()
	if q := strings.TrimSpace(query.Get("q")); q != "" {
		pattern := likePattern(q)
		where += ` AND (LOWER(a.title) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(ar.name) LIKE LOWER(?) ESCAPE '\')`
		args = append(args, pattern, pattern)
	}
	if letter := strings.TrimSpace(query.Get("letter")); letter != "" {
		parsed, err := parseLetter(letter)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		condition, value := musicLetterCondition("a.title", parsed)
		where += " AND " + condition
		if value != nil {
			args = append(args, value)
		}
	}
	if raw := strings.TrimSpace(query.Get("artist")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeJSONError(w, http.StatusBadRequest, "artist debe ser un entero positivo")
			return
		}
		where += " AND a.artist_id = ?"
		args = append(args, id)
	}
	order := " ORDER BY a.title COLLATE NOCASE, a.id"
	switch strings.TrimSpace(query.Get("sort")) {
	case "", "title":
	case "artist":
		order = " ORDER BY ar.sort_name COLLATE NOCASE, a.title COLLATE NOCASE, a.id"
	case "year":
		order = " ORDER BY (a.year IS NULL) ASC, a.year DESC, a.title COLLATE NOCASE, a.id"
	case "added":
		order = " ORDER BY a.created_at DESC, a.id DESC"
	default:
		writeJSONError(w, http.StatusBadRequest, "sort debe ser title, artist, year o added")
		return
	}
	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM music_albums a JOIN music_artists ar ON ar.id = a.artist_id WHERE `+where,
		args...).Scan(&total); err != nil {
		log.Printf("library: contar álbumes: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	albums, _, err := queryMusicAlbums(r.Context(), a.db, where, args, order, limit, offset)
	if err != nil {
		log.Printf("library: listar álbumes: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, musicAlbumsResponse{
		Albums: albums, Total: total, Limit: limit, Offset: offset,
	})
}

const musicAlbumColumns = `
	a.id, a.title, ar.name, a.artist_id, a.year,
	(SELECT count(*) FROM music_tracks mt WHERE mt.album_id = a.id),
	COALESCE((SELECT SUM(mt.duration_seconds) FROM music_tracks mt WHERE mt.album_id = a.id), 0),
	CASE WHEN (a.cover_path IS NOT NULL AND TRIM(a.cover_path) <> '') THEN 1
	     WHEN EXISTS (SELECT 1 FROM music_tracks mt WHERE mt.album_id = a.id AND mt.has_embedded_cover = 1) THEN 1
	     ELSE 0 END`

func queryMusicAlbums(
	ctx context.Context,
	db *sql.DB,
	where string,
	args []any,
	order string,
	limit, offset int,
) ([]musicAlbumDTO, int64, error) {
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := db.QueryContext(ctx, `
		SELECT `+musicAlbumColumns+`
		  FROM music_albums a
		  JOIN music_artists ar ON ar.id = a.artist_id
		 WHERE `+where+order+` LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	albums := make([]musicAlbumDTO, 0)
	for rows.Next() {
		album, err := scanMusicAlbum(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		albums = append(albums, album)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM music_albums a JOIN music_artists ar ON ar.id = a.artist_id WHERE `+where,
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return albums, total, nil
}

func scanMusicAlbum(scan func(dest ...any) error) (musicAlbumDTO, error) {
	var album musicAlbumDTO
	var year sql.NullInt64
	var hasCover int
	if err := scan(&album.ID, &album.Title, &album.Artist, &album.ArtistID, &year,
		&album.TrackCount, &album.DurationSeconds, &hasCover); err != nil {
		return musicAlbumDTO{}, err
	}
	album.Year = intPointer(year)
	album.HasCover = hasCover != 0
	return album, nil
}

func (a *API) handleMusicAlbum(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "álbum no encontrado")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	album, err := loadMusicAlbum(r.Context(), a.db, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "álbum no encontrado")
		return
	case err != nil:
		log.Printf("library: leer álbum %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM music_tracks mt WHERE mt.album_id = ?`, id).Scan(&total); err != nil {
		log.Printf("library: contar pistas del álbum %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	tracks, err := queryMusicTracks(r.Context(), a.db, "mt.album_id = ?", []any{id},
		" ORDER BY mt.disc_no, mt.track_no IS NULL, mt.track_no, mt.title COLLATE NOCASE, mt.id", limit, offset)
	if err != nil {
		log.Printf("library: listar pistas del álbum %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, musicAlbumDetailDTO{
		musicAlbumDTO: album, Tracks: tracks, Total: total, Limit: limit, Offset: offset,
	})
}

func loadMusicAlbum(ctx context.Context, db *sql.DB, id int64) (musicAlbumDTO, error) {
	row := db.QueryRowContext(ctx, `
		SELECT `+musicAlbumColumns+`
		  FROM music_albums a
		  JOIN music_artists ar ON ar.id = a.artist_id
		 WHERE a.id = ?`, id)
	return scanMusicAlbum(row.Scan)
}

func (a *API) handleMusicTracks(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	where := "1 = 1"
	args := make([]any, 0)
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		pattern := likePattern(q)
		where += ` AND (LOWER(mt.title) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(ar.name) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(a.title) LIKE LOWER(?) ESCAPE '\')`
		args = append(args, pattern, pattern, pattern)
	}
	var total int64
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT count(*) FROM music_tracks mt
		  JOIN music_albums a ON a.id = mt.album_id
		  JOIN music_artists ar ON ar.id = mt.artist_id
		 WHERE `+where, args...).Scan(&total); err != nil {
		log.Printf("library: contar pistas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	tracks, err := queryMusicTracks(r.Context(), a.db, where, args,
		" ORDER BY ar.sort_name COLLATE NOCASE, a.title COLLATE NOCASE, mt.disc_no, mt.track_no, mt.title COLLATE NOCASE, mt.id",
		limit, offset)
	if err != nil {
		log.Printf("library: buscar pistas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, musicTracksResponse{
		Tracks: tracks, Total: total, Limit: limit, Offset: offset,
	})
}

func queryMusicTracks(
	ctx context.Context,
	db *sql.DB,
	where string,
	args []any,
	order string,
	limit, offset int,
) ([]musicTrackDTO, error) {
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := db.QueryContext(ctx, `
		SELECT mt.id, mt.title, mt.track_no, mt.disc_no, ar.name,
		       mt.duration_seconds, mt.codec, mt.bitrate, mt.channels, mt.path
		  FROM music_tracks mt
		  JOIN music_albums a ON a.id = mt.album_id
		  JOIN music_artists ar ON ar.id = mt.artist_id
		 WHERE `+where+order+` LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := make([]musicTrackDTO, 0)
	for rows.Next() {
		track, err := scanMusicTrack(rows.Scan)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

func scanMusicTrack(scan func(dest ...any) error) (musicTrackDTO, error) {
	var track musicTrackDTO
	var trackNo, discNo sql.NullInt64
	var duration sql.NullFloat64
	var codec sql.NullString
	var bitrate, channels sql.NullInt64
	var path string
	if err := scan(&track.ID, &track.Title, &trackNo, &discNo, &track.Artist,
		&duration, &codec, &bitrate, &channels, &path); err != nil {
		return musicTrackDTO{}, err
	}
	track.TrackNo = intPointer(trackNo)
	if discNo.Valid {
		track.DiscNo = int(discNo.Int64)
	} else {
		track.DiscNo = 1
	}
	track.DurationSeconds = floatPointer(duration)
	track.Codec = stringPointer(codec)
	track.BitRate = int64Pointer(bitrate)
	channelsValue := 0
	if channels.Valid {
		channelsValue = int(channels.Int64)
	}
	track.Playable = audioPlayable(path, channelsValue)
	return track, nil
}

func (a *API) handleMusicTrackPlay(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "pista no encontrada")
		return
	}
	var path, libraryRoot string
	var libraryID int64
	var channels sql.NullInt64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mt.path, mt.channels, l.id, l.path
		  FROM music_tracks mt
		  JOIN libraries l ON l.id = mt.library_id
		 WHERE mt.id = ? AND l.kind = 'music'`, id).
		Scan(&path, &channels, &libraryID, &libraryRoot)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "pista no encontrada")
		return
	case err != nil:
		log.Printf("library: leer pista %d para reproducir: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	libraryRoot = a.libraryRootFor(r.Context(), libraryID, path, libraryRoot)
	safePath, err := playback.SafePath(libraryRoot, path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	if stat, err := os.Stat(safePath); err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	if a.playback == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "reproducción no disponible")
		return
	}
	channelsValue := 0
	if channels.Valid {
		channelsValue = int(channels.Int64)
	}
	signature, expiresAt := a.playback.SignResource("music", id)
	expires := strconv.FormatInt(expiresAt.Unix(), 10)
	url := fmt.Sprintf("/api/v1/stream/music/%d?exp=%s&sig=%s", id, expires, signature)
	writeJSON(w, http.StatusOK, musicPlayResponse{
		URL:       url,
		MIME:      audioMIME(path),
		ExpiresAt: expiresAt,
		Playable:  audioPlayable(path, channelsValue),
	})
}

func (a *API) handleMusicStream(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "pista no encontrada")
		return
	}
	if a.playback == nil {
		writeJSONError(w, http.StatusNotFound, "pista no encontrada")
		return
	}
	exp := r.URL.Query().Get("exp")
	signature := r.URL.Query().Get("sig")
	if err := a.playback.VerifyResource("music", id, exp, signature); err != nil {
		if errors.Is(err, playback.ErrExpired) {
			writeJSONError(w, http.StatusForbidden, "URL de reproducción caducada")
		} else {
			writeJSONError(w, http.StatusForbidden, "firma de reproducción no válida")
		}
		return
	}
	var path, libraryRoot string
	var libraryID int64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mt.path, l.id, l.path
		  FROM music_tracks mt
		  JOIN libraries l ON l.id = mt.library_id
		 WHERE mt.id = ? AND l.kind = 'music'`, id).
		Scan(&path, &libraryID, &libraryRoot)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "pista no encontrada")
		return
	}
	if err != nil {
		log.Printf("library: leer pista %d para servir: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	libraryRoot = a.libraryRootFor(r.Context(), libraryID, path, libraryRoot)
	safePath, err := playback.SafePath(libraryRoot, path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	file, err := os.Open(safePath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	w.Header().Set("Content-Type", audioMIME(path))
	http.ServeContent(w, r, filepath.Base(safePath), stat.ModTime(), file)
}

func (a *API) handleMusicAlbumCover(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "álbum no encontrado")
		return
	}
	var coverPath, libraryRoot string
	var libraryID int64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT COALESCE(a.cover_path, ''), a.library_id, l.path
		  FROM music_albums a
		  JOIN libraries l ON l.id = a.library_id
		 WHERE a.id = ?`, id).
		Scan(&coverPath, &libraryID, &libraryRoot)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "álbum no encontrado")
		return
	case err != nil:
		log.Printf("library: leer carátula del álbum %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if coverPath != "" {
		libraryRoot = a.libraryRootFor(r.Context(), libraryID, coverPath, libraryRoot)
		if safeCover, err := playback.SafePath(libraryRoot, coverPath); err == nil {
			if stat, err := os.Stat(safeCover); err == nil && !stat.IsDir() {
				serveMusicImage(w, r, safeCover)
				return
			}
		}
	}

	var trackPath string
	err = a.db.QueryRowContext(r.Context(), `
		SELECT path FROM music_tracks
		 WHERE album_id = ? AND has_embedded_cover = 1
		 ORDER BY disc_no, track_no IS NULL, track_no, id LIMIT 1`, id).
		Scan(&trackPath)
	if errors.Is(err, sql.ErrNoRows) || trackPath == "" {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	if err != nil {
		log.Printf("library: buscar carátula incrustada del álbum %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	libraryRoot = a.libraryRootFor(r.Context(), libraryID, trackPath, libraryRoot)
	safePath, err := playback.SafePath(libraryRoot, trackPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	if a.musicCovers == nil {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	cover, err := a.musicCovers.Get(r.Context(), id, safePath, stat.ModTime().UnixNano())
	if err != nil {
		if errors.Is(err, ErrThumbnailBusy) {
			w.Header().Set("Retry-After", "5")
			writeJSONError(w, http.StatusServiceUnavailable, "carátula ocupada, inténtalo de nuevo")
			return
		}
		log.Printf("library: carátula incrustada del álbum %d: %v", id, err)
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	serveMusicImage(w, r, cover)
}

func serveMusicImage(w http.ResponseWriter, r *http.Request, path string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	etag := fmt.Sprintf(`"%x-%x"`, info.ModTime().UnixNano(), info.Size())
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", imageMIME(path))
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "carátula no encontrada")
		return
	}
	defer file.Close()
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}
