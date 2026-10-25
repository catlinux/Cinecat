// Comando cinecatd: servidor de Cinecat.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	ctx := context.Background()
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
	if cfg.OMDbAPIKey != "" {
		omdb, err := metadata.NewOMDbClient(metadata.OMDbOptions{APIKey: cfg.OMDbAPIKey})
		if err != nil {
			log.Printf("cinecatd: metadatos de OMDb desactivados: %v", err)
		} else {
			scanner = scanner.WithOMDb(omdb)
		}
	}

	api := library.NewAPI(st.DB(), authSvc, playSvc, scanner)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           newHandler(authSvc, api),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("cinecatd %s escuchando en %s", applicationVersion(), cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("cinecatd: %v", err)
	}
}

func newHandler(authSvc *auth.Service, api *library.API) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
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
