package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvConfigFile, "")
	t.Setenv(EnvListenAddr, "")
	t.Setenv(EnvTMDBAPIKey, "")
	t.Setenv(EnvOMDbAPIKey, "")
	t.Setenv(EnvScanInterval, "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Fatalf("ListenAddr = %q, se esperaba %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.ScanInterval != DefaultScanInterval {
		t.Fatalf("ScanInterval = %s, se esperaba %s", cfg.ScanInterval, DefaultScanInterval)
	}
	if cfg.TMDBAPIKey != "" {
		t.Fatalf("TMDBAPIKey = %q, se esperaba vacía por defecto", cfg.TMDBAPIKey)
	}
	if cfg.OMDbAPIKey != "" {
		t.Fatalf("OMDbAPIKey = %q, se esperaba vacía por defecto", cfg.OMDbAPIKey)
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

func TestLoadClaveOMDb(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"omdb_api_key":"desde-archivo"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvOMDbAPIKey, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OMDbAPIKey != "desde-archivo" {
		t.Fatalf("OMDbAPIKey = %q, se esperaba la del archivo", cfg.OMDbAPIKey)
	}

	t.Setenv(EnvOMDbAPIKey, "desde-entorno")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load con entorno: %v", err)
	}
	if cfg.OMDbAPIKey != "desde-entorno" {
		t.Fatalf("OMDbAPIKey = %q, se esperaba la del entorno", cfg.OMDbAPIKey)
	}
}

func TestLoadArchivoInexistente(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "no-existe.json")); err == nil {
		t.Fatal("se esperaba error con un archivo inexistente")
	}
}

func TestLoadIntervaloEscaneo(t *testing.T) {
	t.Setenv(EnvConfigFile, "")
	t.Setenv(EnvScanInterval, "0")
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("intervalo 0: %v", err)
	}
	if cfg.ScanInterval != 0 {
		t.Fatalf("ScanInterval = %s, se esperaba 0", cfg.ScanInterval)
	}

	t.Setenv(EnvScanInterval, "90m")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("intervalo válido: %v", err)
	}
	if cfg.ScanInterval != 90*time.Minute {
		t.Fatalf("ScanInterval = %s, se esperaba 90m", cfg.ScanInterval)
	}

	for _, invalid := range []string{"no-es-duracion", "-1h"} {
		t.Setenv(EnvScanInterval, invalid)
		if _, err := Load(""); err == nil {
			t.Fatalf("%q debe dar error de validación", invalid)
		}
	}
}

func TestLoadIntervaloEscaneoDesdeArchivo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"scan_interval":"30m"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvScanInterval, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ScanInterval != 30*time.Minute {
		t.Fatalf("ScanInterval = %s, se esperaba 30m", cfg.ScanInterval)
	}

	t.Setenv(EnvScanInterval, "2h")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load con entorno: %v", err)
	}
	if cfg.ScanInterval != 2*time.Hour {
		t.Fatalf("ScanInterval = %s, se esperaba la del entorno", cfg.ScanInterval)
	}
}
