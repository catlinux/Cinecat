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
	playSvc, err := playback.New(playback.Options{})
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
	go scheduleScans(ctx, scanJob, time.Minute, cfg.ScanInterval)

	api := library.NewAPI(st.DB(), authSvc, playSvc, scanJob, tmdbClient)

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

// scheduleScans lanza un escaneo tras el retardo inicial y después cada
// intervalo. Con intervalo 0 no hace nada.
func scheduleScans(ctx context.Context, job scanStarter, initialDelay, interval time.Duration) {
	if interval <= 0 {
		return
	}

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			job.Start(ctx)
			timer.Reset(interval)
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
