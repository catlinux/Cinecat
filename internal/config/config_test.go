package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvConfigFile, "")
	t.Setenv(EnvListenAddr, "")
	t.Setenv(EnvTMDBAPIKey, "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Fatalf("ListenAddr = %q, se esperaba %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.TMDBAPIKey != "" {
		t.Fatalf("TMDBAPIKey = %q, se esperaba vacía por defecto", cfg.TMDBAPIKey)
	}
}

func TestLoadFileYEntorno(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"listen_addr":":9000"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvListenAddr, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != ":9000" {
		t.Fatalf("ListenAddr = %q, se esperaba \":9000\"", cfg.ListenAddr)
	}

	t.Setenv(EnvListenAddr, "127.0.0.1:9100")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load con entorno: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9100" {
		t.Fatalf("ListenAddr = %q, se esperaba la del entorno", cfg.ListenAddr)
	}
}

func TestLoadClaveTMDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"tmdb_api_key":"desde-archivo"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvTMDBAPIKey, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBAPIKey != "desde-archivo" {
		t.Fatalf("TMDBAPIKey = %q, se esperaba la del archivo", cfg.TMDBAPIKey)
	}

	t.Setenv(EnvTMDBAPIKey, "desde-entorno")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load con entorno: %v", err)
	}
	if cfg.TMDBAPIKey != "desde-entorno" {
		t.Fatalf("TMDBAPIKey = %q, se esperaba la del entorno", cfg.TMDBAPIKey)
	}
}

func TestLoadArchivoInexistente(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "no-existe.json")); err == nil {
		t.Fatal("se esperaba error con un archivo inexistente")
	}
}
