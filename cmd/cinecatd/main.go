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
	"github.com/catlinux/cinecat/internal/config"
	"github.com/catlinux/cinecat/internal/library"
	"github.com/catlinux/cinecat/internal/metadata"
	"github.com/catlinux/cinecat/internal/playback"
	"github.com/catlinux/cinecat/internal/settings"
	"github.com/catlinux/cinecat/internal/store"
)

// version se puede fijar en tiempo de compilación con
// -ldflags "-X main.version=<versión>"; si no, se lee del archivo VERSION.
var version string

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

	authSvc := auth.New(st.DB())
	playSvc, err := playback.New(playback.Options{
		Settings: func() playback.TranscodeSettings {
			return playback.TranscodeSettings{
				MaxHeight: settingsStore.Int("playback.max_height"),
				CRF:       settingsStore.Int("playback.crf"),
				Preset:    settingsStore.String("playback.preset"),
			}
		},
		MaxTranscodesFunc: func() int {
			return settingsStore.Int("playback.max_transcodes")
		},
	})
	if err != nil {
		log.Fatalf("cinecatd: %v", err)
	}
	defer playSvc.Close()

	scanner := library.New(st.DB())
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
	go scheduleScans(ctx, scanJob, time.Minute, newStoreScanSettings(settingsStore))

	api := library.NewAPI(st.DB(), authSvc, playSvc, scanJob, tmdbClient)
	api = api.WithSettings(settingsStore)
	api = api.WithFileInspector(playSvc)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           newHandler(authSvc, api),
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

func newHandler(authSvc *auth.Service, api *library.API) http.Handler {
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
	}
	if api != nil {
		api.Register(mux)
	}
	return mux
}

type versionResponse struct {
	Server  string `json:"server"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(versionResponse{
		Server:  applicationVersion(),
		Commit:  commit,
		BuiltAt: builtAt,
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
