package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

type recordingMusicInspector struct {
	mu     sync.Mutex
	infos  map[string]*playback.AudioInfo
	errs   map[string]error
	calls  map[string]int
	active int
	max    int
	delay  time.Duration
}

func (f *recordingMusicInspector) InspectAudio(ctx context.Context, path string) (*playback.AudioInfo, error) {
	f.mu.Lock()
	f.calls[path]++
	f.active++
	if f.active > f.max {
		f.max = f.active
	}
	info := f.infos[path]
	err := f.errs[path]
	delay := f.delay
	f.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	f.mu.Lock()
	f.active--
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if info == nil {
		return &playback.AudioInfo{}, nil
	}
	copied := *info
	return &copied, nil
}

func (f *recordingMusicInspector) callCount(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[path]
}

func (f *recordingMusicInspector) maxActive() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.max
}

func TestScannerMusicaEstructuraEtiquetasYCaratula(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	albumDir := filepath.Join(root, "Artista", "Álbum")
	first := filepath.Join(albumDir, "01 - Primera.mp3")
	second := filepath.Join(albumDir, "02 - Segunda.flac")
	writeFile(t, first)
	writeFile(t, second)
	if err := os.WriteFile(filepath.Join(albumDir, "FOLDER.JPG"), []byte("caratula"), 0o644); err != nil {
		t.Fatalf("crear carátula: %v", err)
	}
	res, err := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, libraryID, root)

	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}}
	fake.infos[first] = &playback.AudioInfo{
		Tags: playback.AudioTags{
			Title: "Primera", Artist: "Artista de pista", AlbumArtist: "Various Artists",
			Album: "Álbum", Genre: "Pop", Track: 1, Disc: 2, Year: 2019,
		},
		DurationSeconds: 180, Codec: "mp3", BitRate: 320000, SampleRate: 44100, Channels: 2,
		HasEmbeddedCover: true,
	}
	fake.infos[second] = &playback.AudioInfo{
		Tags:  playback.AudioTags{Title: "Segunda", AlbumArtist: "Varios artistas", Album: "Álbum", Track: 2},
		Codec: "flac", Channels: 2,
	}

	result, err := New(db).WithMusicInspector(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Files != 2 || result.Upserted != 2 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	var artist, album, cover string
	var embedded int
	if err := db.QueryRow(`
		SELECT ar.name, a.title, COALESCE(a.cover_path, ''), mt.has_embedded_cover
		  FROM music_tracks mt
		  JOIN music_albums a ON a.id = mt.album_id
		  JOIN music_artists ar ON ar.id = a.artist_id
		 WHERE mt.path = ?`, first).Scan(&artist, &album, &cover, &embedded); err != nil {
		t.Fatalf("leer pista: %v", err)
	}
	if artist != "Varios artistas" || album != "Álbum" || embedded != 1 {
		t.Fatalf("metadatos: artist=%q album=%q embedded=%d", artist, album, embedded)
	}
	if filepath.Base(cover) != "FOLDER.JPG" {
		t.Fatalf("carátula de carpeta = %q", cover)
	}
	var media int
	if err := db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&media); err != nil || media != 0 {
		t.Fatalf("música no debe crear media_items: %d %v", media, err)
	}
}

func TestScannerMusicaNoReinspeccionaSinCambiosYCambiaArchivo(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	file := filepath.Join(root, "Artista", "Álbum", "01 - Tema.mp3")
	writeFile(t, file)
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)

	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}}
	scanner := New(db).WithMusicInspector(fake)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	if got := fake.callCount(file); got != 1 {
		t.Fatalf("llamadas = %d, se esperaba 1", got)
	}
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if got := fake.callCount(file); got != 1 {
		t.Fatalf("sin cambios no debe reinspeccionar: %d", got)
	}

	if err := os.WriteFile(file, []byte("contenido distinto"), 0o644); err != nil {
		t.Fatalf("cambiar archivo: %v", err)
	}
	newTime := time.Now().Add(3 * time.Second)
	if err := os.Chtimes(file, newTime, newTime); err != nil {
		t.Fatalf("cambiar mtime: %v", err)
	}
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("tercer Scan: %v", err)
	}
	if got := fake.callCount(file); got != 2 {
		t.Fatalf("un archivo cambiado debe reinspeccionarse: %d", got)
	}
}

func TestScannerMusicaPodaYConservaCarpetaVacia(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	keep := filepath.Join(root, "Artista A", "Álbum A", "01 - Uno.mp3")
	remove := filepath.Join(root, "Artista B", "Álbum B", "01 - Dos.mp3")
	writeFile(t, keep)
	writeFile(t, remove)
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)
	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}}
	scanner := New(db).WithMusicInspector(fake)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan inicial: %v", err)
	}

	if err := os.Remove(remove); err != nil {
		t.Fatalf("retirar pista: %v", err)
	}
	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan tras retirar: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM music_tracks`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("pistas tras poda = %d, %v", count, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM music_artists WHERE name = 'Artista B'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("el artista huérfano debe borrarse: %d, %v", count, err)
	}

	for _, path := range []string{keep, remove} {
		_ = os.Remove(path)
	}
	result, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan con carpeta vacía: %v", err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM music_tracks`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("la carpeta vacía debe conservar pistas: %d, %v", count, err)
	}
	found := false
	for _, warning := range result.Warnings {
		if len(warning) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("debe avisar de la carpeta vacía")
	}
}

func TestScannerMusicaFalloSeIndexaConRespaldo(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	file := filepath.Join(root, "Sin Etiquetas", "02 - Tema.mp3")
	writeFile(t, file)
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)
	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}, errs: map[string]error{file: errors.New("ffprobe roto")}}
	result, err := New(db).WithMusicInspector(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("debe avisar del fallo de ffprobe")
	}
	var title, album, artist string
	if err := db.QueryRow(`
		SELECT mt.title, ma.title, ar.name
		  FROM music_tracks mt
		  JOIN music_albums ma ON ma.id = mt.album_id
		  JOIN music_artists ar ON ar.id = mt.artist_id`).Scan(&title, &album, &artist); err != nil {
		t.Fatalf("leer pista: %v", err)
	}
	if title != "Tema" || album != "Sin Etiquetas" || artist != unknownMusicArtist {
		t.Fatalf("respaldo inesperado: title=%q album=%q artist=%q", title, album, artist)
	}
}

func TestScannerMusicaLimitaConcurrencia(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)
	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}, delay: 20 * time.Millisecond}
	for i := 0; i < 12; i++ {
		writeFile(t, filepath.Join(root, "Álbum", strconv.Itoa(i)+".mp3"))
	}
	if _, err := New(db).WithMusicInspector(fake).Scan(ctx); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if fake.maxActive() > 4 || fake.maxActive() < 2 {
		t.Fatalf("concurrencia máxima = %d", fake.maxActive())
	}
}

func TestScannerMusicaVariasCarpetas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	first := filepath.Join(rootA, "Artista A", "Álbum A", "01.mp3")
	second := filepath.Join(rootB, "Artista B", "Álbum B", "01.mp3")
	writeFile(t, first)
	writeFile(t, second)
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, rootA, "Música")
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, rootA)
	addScannerLibraryPath(t, db, id, rootB)
	fake := &recordingMusicInspector{infos: map[string]*playback.AudioInfo{}, calls: map[string]int{}}
	result, err := New(db).WithMusicInspector(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Files != 2 {
		t.Fatalf("archivos = %d, se esperaban 2", result.Files)
	}
	var artists int
	if err := db.QueryRow(`SELECT count(*) FROM music_artists`).Scan(&artists); err != nil || artists != 2 {
		t.Fatalf("artistas = %d, %v", artists, err)
	}
}
