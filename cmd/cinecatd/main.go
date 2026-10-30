// Comando cinecatd: servidor de Cinecat.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/backup"
	"github.com/catlinux/cinecat/internal/config"
	"github.com/catlinux/cinecat/internal/library"
	"github.com/catlinux/cinecat/internal/metadata"
	"github.com/catlinux/cinecat/internal/playback"
	"github.com/catlinux/cinecat/internal/settings"
	"github.com/catlinux/cinecat/internal/store"
	"github.com/catlinux/cinecat/internal/update"
)

// version se puede fijar en tiempo de compilación con
// -ldflags "-X main.version=<versión>"; si no, se lee del archivo VERSION.
var version string

// apiLevel solo cambia cuando una ruta de /api/v1 se modifica de forma
// incompatible; minClient es la versión mínima de app nativa admitida.
const (
	apiLevel  = 1
	minClient = "0.0.0"
)

// commit y builtAt se inyectan con -ldflags "-X main.commit=... -X main.builtAt=...".
var (
	commit  string
	builtAt string
)

func main() {
	showVersion := flag.Bool("version", false, "imprime la versión y sale")
	hashPassword := flag.String("hash-password", "", "genera el hash argon2id de la contraseña indicada y sale (uso: actualizar password_hash a mano)")
	configPath := flag.String("config", "", "ruta al archivo de configuración (fuera del repositorio)")
	flag.Parse()

	if *showVersion {
		fmt.Println(applicationVersion())
		return
	}

	if *hashPassword != "" {
		hash, err := auth.HashPassword(*hashPassword)
		if err != nil {
			log.Fatalf("cinecatd: %v", err)
		}
		fmt.Println(hash)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	signingKey, err := loadSigningKey(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}

	st, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("cinecatd: %v", err)
	}
	if err := library.BackfillSortKeys(ctx, st.DB()); err != nil {
		log.Printf("cinecatd: títulos normalizados pendientes: %v", err)
	}

	settingsStore := settings.NewStore(st.DB(), map[string]any{
		settings.KeyScanInterval: cfg.ScanInterval,
	})
	if err := settingsStore.Load(ctx); err != nil {
		log.Printf("cinecatd: no se pudieron cargar los ajustes guardados: %v", err)
	}
	library.SetMatchThreshold(settingsStore.Float("metadata.match_threshold"))
	settingsStore.OnChange(func(keys []string) {
		for _, key := range keys {
			if key == "metadata.match_threshold" {
				library.SetMatchThreshold(settingsStore.Float(key))
				return
			}
		}
	})

	bootstrap, err := auth.BootstrapAdmin(ctx, st.DB(), os.Getenv(auth.EnvAdminPassword))
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}
	if bootstrap.Created {
		if bootstrap.GeneratedPassword != "" {
			log.Printf("cinecatd: creado el usuario administrador %q con esta contraseña temporal (se muestra solo esta vez): %s",
				bootstrap.Username, bootstrap.GeneratedPassword)
			log.Printf("cinecatd: cambia la contraseña del administrador en cuanto inicies sesión")
		} else {
			log.Printf("cinecatd: creado el usuario administrador %q con la contraseña de %s",
				bootstrap.Username, auth.EnvAdminPassword)
		}
	}

	authSvc := auth.New(st.DB(), auth.WithTrustedProxies(cfg.TrustedProxies))
	go cleanupPairings(ctx, authSvc)
	playSvc, err := playback.New(playback.Options{
		Settings: func() playback.TranscodeSettings {
			return playback.TranscodeSettings{
				MaxHeight: settingsStore.Int("playback.max_height"),
				CRF:       settingsStore.Int("playback.crf"),
				Preset:    settingsStore.String("playback.preset"),
				Hardware:  settingsStore.String(settings.KeyPlaybackHardware) != "off",
			}
		},
		MaxTranscodesFunc: func() int {
			return settingsStore.Int("playback.max_transcodes")
		},
		KeyframeCacheDir: keyframeCacheDir(cfg.DatabasePath),
		Key:              signingKey,
	})
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}
	defer playSvc.Close()

	scanner := library.New(st.DB()).WithMusicInspector(playSvc)
	var tmdbClient library.TMDBClient
	if cfg.TMDBAPIKey != "" {
		tmdb, err := metadata.New(metadata.Options{
			APIKey:   cfg.TMDBAPIKey,
			CacheDir: tmdbCacheDir(cfg.DatabasePath),
			Language: func() string { return settingsStore.String("metadata.language") },
		})
		if err != nil {
			log.Printf("cinecatd: metadatos de TMDB desactivados: %v", err)
		} else {
			scanner = scanner.WithTMDB(tmdb)
			tmdbClient = tmdb
		}
	} else if cfg.OMDbAPIKey != "" {
		omdb, err := metadata.NewOMDbClient(metadata.OMDbOptions{APIKey: cfg.OMDbAPIKey})
		if err != nil {
			log.Printf("cinecatd: metadatos de OMDb desactivados: %v", err)
		} else {
			scanner = scanner.WithOMDb(omdb)
		}
	}

	scanJob := library.NewScanJob(scanner)
	defer scanJob.Close()
	if watchEnabled() {
		watcher := library.NewLibraryWatcher(st.DB(), scanJob, watchQuiet(), 10*time.Minute)
		go watcher.Run(ctx)
	}
	go scheduleScans(ctx, scanJob, time.Minute, newStoreScanSettings(settingsStore))

	updateChecker := update.NewChecker(update.CheckerOptions{Current: applicationVersion()})
	updateRequester := update.NewRequester(os.Getenv(update.EnvRunDir))
	updateChanges := make(chan struct{}, 1)
	settingsStore.OnChange(func(keys []string) {
		for _, key := range keys {
			if key == "system.check_updates" {
				select {
				case updateChanges <- struct{}{}:
				default:
				}
				return
			}
		}
	})
	go update.RunBackground(
		ctx,
		updateChecker,
		func() bool { return settingsStore.Bool("system.check_updates") },
		updateChanges,
		2*time.Minute,
		6*time.Hour,
	)

	backupManager := backup.New(st.DB(), cfg.DatabasePath).WithSettings(
		func() bool { return settingsStore.Bool(settings.KeyBackupEnabled) },
		func() int { return settingsStore.Int(settings.KeyBackupRetentionDays) },
	)
	backupChanges := make(chan struct{}, 1)
	settingsStore.OnChange(func(keys []string) {
		for _, key := range keys {
			if key == settings.KeyBackupEnabled || key == settings.KeyBackupRetentionDays {
				select {
				case backupChanges <- struct{}{}:
				default:
				}
				return
			}
		}
	})
	go backup.RunBackground(ctx, backupManager, backupChanges, 2*time.Minute, 24*time.Hour, time.Hour)

	api := library.NewAPI(st.DB(), authSvc, playSvc, scanJob, tmdbClient)
	api = api.WithSettings(settingsStore)
	api = api.WithFileInspector(playSvc)
	api = api.WithThumbnails(thumbnailCacheDir(cfg.DatabasePath))
	api = api.WithMusicCovers(musicCoverCacheDir(cfg.DatabasePath))
	api = api.WithTMDBImages(tmdbImageCacheDir(cfg.DatabasePath))
	api = api.WithMediaRoots(cfg.MediaRoots)
	go api.StartTrashPurger(ctx)

	updateAPI := update.NewAPI(
		authSvc.Authenticate,
		updateChecker,
		updateRequester,
		func() bool { return settingsStore.Bool("system.check_updates") },
	)
	backupAPI := backup.NewAPI(
		authSvc.Authenticate,
		backupManager,
		func() bool { return settingsStore.Bool(settings.KeyBackupEnabled) },
		func() int { return settingsStore.Int(settings.KeyBackupRetentionDays) },
	)
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           newHandlerWithBackup(authSvc, api, updateAPI, backupAPI),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("cinecatd %s escuchando en %s", applicationVersion(), cfg.ListenAddr)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("cinecatd: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("cinecatd: apagado: %v", err)
		}
	}
}

type scanStarter interface {
	Start(context.Context) bool
}

type scanSettings interface {
	Interval() time.Duration
	OnStartup() bool
	Changes() <-chan struct{}
}

// storeScanSettings adapta el Store de ajustes al planificador. El canal
// despierta el bucle para recalcular el siguiente escaneo sin esperar al
// temporizador anterior.
type storeScanSettings struct {
	store   *settings.Store
	changes chan struct{}
}

func newStoreScanSettings(store *settings.Store) *storeScanSettings {
	settings := &storeScanSettings{
		store:   store,
		changes: make(chan struct{}, 1),
	}
	if store != nil {
		store.OnChange(func(keys []string) {
			for _, key := range keys {
				if key != "scan.interval" && key != "scan.on_startup" {
					continue
				}
				select {
				case settings.changes <- struct{}{}:
				default:
				}
				break
			}
		})
	}
	return settings
}

func (s *storeScanSettings) Interval() time.Duration {
	if s == nil || s.store == nil {
		return 0
	}
	return s.store.Duration("scan.interval")
}

func (s *storeScanSettings) OnStartup() bool {
	if s == nil || s.store == nil {
		return false
	}
	return s.store.Bool("scan.on_startup")
}

func (s *storeScanSettings) Changes() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.changes
}

// scheduleScans lanza un escaneo tras el retardo inicial si está activado y
// después cada intervalo. Los cambios de ajustes recalculan el siguiente
// instante; con intervalo 0 el planificador espera sin lanzar escaneos.
func scheduleScans(ctx context.Context, job scanStarter, initialDelay time.Duration, settings scanSettings) {
	if settings == nil {
		return
	}

	var timer *time.Timer
	var timerC <-chan time.Time

	stopTimer := func() {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer = nil
		timerC = nil
	}
	resetTimer := func(delay time.Duration) {
		stopTimer()
		if delay < 0 {
			return
		}
		timer = time.NewTimer(delay)
		timerC = timer.C
	}

	startupHandled := false
	scheduleNext := func() {
		interval := settings.Interval()
		if interval <= 0 {
			startupHandled = true
			stopTimer()
			return
		}
		if !startupHandled && settings.OnStartup() {
			resetTimer(initialDelay)
			return
		}
		startupHandled = true
		resetTimer(interval)
	}

	scheduleNext()
	defer stopTimer()

	for {
		select {
		case <-ctx.Done():
			return
		case <-settings.Changes():
			scheduleNext()
		case <-timerC:
			job.Start(ctx)
			startupHandled = true
			if interval := settings.Interval(); interval > 0 {
				resetTimer(interval)
			} else {
				stopTimer()
			}
		}
	}
}

const (
	envWatch          = "CINECAT_WATCH"
	envWatchQuiet     = "CINECAT_WATCH_QUIET"
	defaultWatchQuiet = 2 * time.Minute
)

func watchEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(envWatch)))
	switch value {
	case "0", "false", "off":
		return false
	default:
		return true
	}
}

func watchQuiet() time.Duration {
	value := strings.TrimSpace(os.Getenv(envWatchQuiet))
	if value == "" {
		return defaultWatchQuiet
	}
	quiet, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("cinecatd: %s no es una duración válida (%q); se usa %s", envWatchQuiet, value, defaultWatchQuiet)
		return defaultWatchQuiet
	}
	return quiet
}

// cleanupPairings borra las vinculaciones caducadas o consumidas hace tiempo.
func cleanupPairings(ctx context.Context, svc *auth.Service) {
	if svc == nil {
		return
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.CleanupPairings(ctx); err != nil {
				log.Printf("cinecatd: limpiar vinculaciones: %v", err)
			}
		}
	}
}

// tmdbCacheDir deja la caché junto a la base de datos para que viva en el
// volumen de Docker y sobreviva a las actualizaciones. Con ":memory:" no hay
// archivo junto al que guardarla.
func tmdbCacheDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "tmdb")
}

// tmdbImageCacheDir guarda las imágenes de TMDB en el volumen de la base.
func tmdbImageCacheDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "images")
}

// thumbnailCacheDir guarda los fotogramas en el volumen de la base, para que
// sobrevivan a las actualizaciones y se puedan borrar sin tocar la biblioteca.
func thumbnailCacheDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "thumbs")
}

// keyframeCacheDir guarda los índices de claves junto a la base de datos.
func keyframeCacheDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "keyframes")
}

// musicCoverCacheDir guarda las carátulas incrustadas extraídas de las pistas.
func musicCoverCacheDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "covers")
}

func newHandler(authSvc *auth.Service, api *library.API, updateAPIs ...*update.API) http.Handler {
	var updateAPI *update.API
	if len(updateAPIs) > 0 {
		updateAPI = updateAPIs[0]
	}
	return newHandlerWithBackup(authSvc, api, updateAPI, nil)
}

func newHandlerWithBackup(
	authSvc *auth.Service,
	api *library.API,
	updateAPI *update.API,
	backupAPI *backup.API,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /api/v1/system/version", handleVersion)
	if authSvc != nil {
		mux.Handle("POST /api/v1/auth/login", authSvc.LoginHandler())
		mux.Handle("POST /api/v1/auth/logout", authSvc.LogoutHandler())
		mux.Handle("POST /api/v1/auth/password", authSvc.ChangePasswordHandler())
		mux.Handle("GET /api/v1/auth/me", authSvc.CurrentUserHandler())
		authSvc.RegisterUserRoutes(mux)
		authSvc.RegisterSessionRoutes(mux)
		authSvc.RegisterPairingRoutes(mux)
	}
	if api != nil {
		api.Register(mux)
	}
	if updateAPI != nil {
		updateAPI.Register(mux)
	}
	if backupAPI != nil {
		backupAPI.Register(mux)
	}
	return securityHeaders(mux)
}

// securityHeaders añade las cabeceras comunes de la API sin envolver el
// ResponseWriter: Range, streaming y Flush siguen funcionando igual.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

type versionResponse struct {
	Server    string `json:"server"`
	Commit    string `json:"commit"`
	BuiltAt   string `json:"built_at"`
	APILevel  int    `json:"api_level"`
	MinClient string `json:"min_client"`
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(versionResponse{
		Server:    applicationVersion(),
		Commit:    commit,
		BuiltAt:   builtAt,
		APILevel:  apiLevel,
		MinClient: minClient,
	}); err != nil {
		log.Printf("cinecatd: escribir versión: %v", err)
	}
}

// applicationVersion devuelve la versión compilada o, si no la hay, el
// contenido del archivo VERSION junto al ejecutable o en el directorio actual.
func applicationVersion() string {
	if v := strings.TrimSpace(version); v != "" {
		return v
	}
	for _, path := range versionFileCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if v := strings.TrimSpace(string(data)); v != "" {
			return v
		}
	}
	return "desconocida"
}

func versionFileCandidates() []string {
	candidates := []string{"VERSION"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "VERSION"))
	}
	return candidates
}
