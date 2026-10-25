// Package config carga la configuración del servidor Cinecat.
//
// La configuración llega siempre desde fuera del binario: un archivo JSON
// indicado con --config/CINECAT_CONFIG (ignorado por git, nunca en el
// repositorio) y variables de entorno CINECAT_*, que tienen prioridad.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Variables de entorno reconocidas. Las claves de API y demás secretos se
// pasan por aquí o por el archivo de configuración local, nunca por el código.
const (
	EnvConfigFile   = "CINECAT_CONFIG"
	EnvListenAddr   = "CINECAT_LISTEN_ADDR"
	EnvDatabasePath = "CINECAT_DB_PATH"
	EnvTMDBAPIKey   = "CINECAT_TMDB_API_KEY"
	EnvOMDbAPIKey   = "CINECAT_OMDB_API_KEY"
)

// Valores por defecto cuando no se configura otra cosa.
const (
	DefaultListenAddr   = ":8080"
	DefaultDatabasePath = "cinecat.db"
)

// Config agrupa los ajustes de arranque del servidor.
type Config struct {
	// ListenAddr es la dirección y puerto donde escucha el servidor HTTP.
	ListenAddr string `json:"listen_addr"`
	// DatabasePath es el archivo SQLite con la base de datos.
	DatabasePath string `json:"database_path"`
	// TMDBAPIKey es la clave de la API de TMDB; se lee de fuera del
	// repositorio y no se valida aquí porque es opcional hasta que se use.
	TMDBAPIKey string `json:"tmdb_api_key,omitempty"`
	// OMDbAPIKey es la clave de la API de OMDb, también opcional y fuera del
	// repositorio.
	OMDbAPIKey string `json:"omdb_api_key,omitempty"`
}

// Default devuelve la configuración por defecto.
func Default() Config {
	return Config{ListenAddr: DefaultListenAddr, DatabasePath: DefaultDatabasePath}
}

// Load construye la configuración a partir de los valores por defecto, del
// archivo indicado (si no está vacío) y de las variables de entorno. El
// argumento configPath tiene prioridad sobre CINECAT_CONFIG.
func Load(configPath string) (Config, error) {
	cfg := Default()

	path := strings.TrimSpace(configPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv(EnvConfigFile))
	}
	if path != "" {
		if err := cfg.mergeFile(path); err != nil {
			return Config{}, err
		}
	}

	if v := strings.TrimSpace(os.Getenv(EnvListenAddr)); v != "" {
		cfg.ListenAddr = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvDatabasePath)); v != "" {
		cfg.DatabasePath = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvTMDBAPIKey)); v != "" {
		cfg.TMDBAPIKey = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvOMDbAPIKey)); v != "" {
		cfg.OMDbAPIKey = v
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) mergeFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: no se puede leer %q: %w", path, err)
	}

	var file Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return fmt.Errorf("config: %q no es un JSON válido: %w", path, err)
	}

	if file.ListenAddr != "" {
		c.ListenAddr = file.ListenAddr
	}
	if file.DatabasePath != "" {
		c.DatabasePath = file.DatabasePath
	}
	if file.TMDBAPIKey != "" {
		c.TMDBAPIKey = file.TMDBAPIKey
	}
	if file.OMDbAPIKey != "" {
		c.OMDbAPIKey = file.OMDbAPIKey
	}
	return nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.ListenAddr) == "" {
		return errors.New("config: listen_addr no puede estar vacío")
	}
	if strings.TrimSpace(c.DatabasePath) == "" {
		return errors.New("config: database_path no puede estar vacío")
	}
	return nil
}
