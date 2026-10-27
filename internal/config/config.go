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
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Variables de entorno reconocidas. Las claves de API y demás secretos se
// pasan por aquí o por el archivo de configuración local, nunca por el código.
const (
	EnvConfigFile     = "CINECAT_CONFIG"
	EnvListenAddr     = "CINECAT_LISTEN_ADDR"
	EnvDatabasePath   = "CINECAT_DB_PATH"
	EnvTMDBAPIKey     = "CINECAT_TMDB_API_KEY"
	EnvOMDbAPIKey     = "CINECAT_OMDB_API_KEY"
	EnvScanInterval   = "CINECAT_SCAN_INTERVAL"
	EnvMediaRoots     = "CINECAT_MEDIA_ROOTS"
	EnvTrustedProxies = "CINECAT_TRUSTED_PROXIES"
)

// Valores por defecto cuando no se configura otra cosa.
const (
	DefaultListenAddr   = ":8080"
	DefaultDatabasePath = "cinecat.db"
	DefaultScanInterval = 6 * time.Hour
	DefaultMediaRoot    = "/media"
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
	// ScanInterval es cada cuánto se escanea sola la biblioteca; 0 lo
	// desactiva. En el archivo JSON se escribe como cadena de duración Go.
	ScanInterval time.Duration `json:"-"`
	// MediaRoots son las carpetas dentro de las que se pueden elegir rutas
	// de biblioteca y navegar desde la administración.
	MediaRoots []string `json:"-"`
	// TrustedProxies son las redes desde las que se acepta X-Forwarded-For.
	TrustedProxies []netip.Prefix `json:"-"`
}

// Default devuelve la configuración por defecto.
func Default() Config {
	return Config{
		ListenAddr:     DefaultListenAddr,
		DatabasePath:   DefaultDatabasePath,
		ScanInterval:   DefaultScanInterval,
		MediaRoots:     []string{DefaultMediaRoot},
		TrustedProxies: defaultTrustedProxies(),
	}
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
	if v := strings.TrimSpace(os.Getenv(EnvScanInterval)); v != "" {
		interval, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("config: %s no es una duración válida: %w", EnvScanInterval, err)
		}
		cfg.ScanInterval = interval
	}
	if raw := strings.TrimSpace(os.Getenv(EnvTrustedProxies)); raw != "" {
		proxies, err := ParseTrustedProxies(raw)
		if err != nil {
			return Config{}, fmt.Errorf("config: %s: %w", EnvTrustedProxies, err)
		}
		cfg.TrustedProxies = proxies
	}
	if raw := os.Getenv(EnvMediaRoots); strings.TrimSpace(raw) != "" {
		roots := make([]string, 0)
		for _, root := range filepath.SplitList(raw) {
			if root = strings.TrimSpace(root); root != "" {
				roots = append(roots, root)
			}
		}
		if len(roots) > 0 {
			cfg.MediaRoots = roots
		}
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type fileConfig struct {
	ListenAddr   string `json:"listen_addr"`
	DatabasePath string `json:"database_path"`
	TMDBAPIKey   string `json:"tmdb_api_key,omitempty"`
	OMDbAPIKey   string `json:"omdb_api_key,omitempty"`
	ScanInterval string `json:"scan_interval,omitempty"`
}

func (c *Config) mergeFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: no se puede leer %q: %w", path, err)
	}

	var file fileConfig
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
	if file.ScanInterval != "" {
		interval, err := time.ParseDuration(file.ScanInterval)
		if err != nil {
			return fmt.Errorf("config: scan_interval no es una duración válida: %w", err)
		}
		c.ScanInterval = interval
	}
	return nil
}

var defaultTrustedProxyCIDRs = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
}

// DefaultTrustedProxies devuelve una copia de los rangos de confianza por
// defecto del servidor.
func DefaultTrustedProxies() []netip.Prefix {
	return defaultTrustedProxies()
}

func defaultTrustedProxies() []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(defaultTrustedProxyCIDRs))
	for _, raw := range defaultTrustedProxyCIDRs {
		prefix, err := netip.ParsePrefix(raw)
		if err == nil {
			prefixes = append(prefixes, prefix.Masked())
		}
	}
	return prefixes
}

// ParseTrustedProxies acepta una lista separada por comas de CIDR o de IP
// sueltas. Una IP suelta se convierte en /32 o /128.
func ParseTrustedProxies(raw string) ([]netip.Prefix, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTrustedProxies(), nil
	}
	prefixes := make([]netip.Prefix, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(item); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		address, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("el valor %q no es una IP ni un CIDR válido", item)
		}
		address = address.Unmap()
		bits := 128
		if address.Is4() {
			bits = 32
		}
		prefixes = append(prefixes, netip.PrefixFrom(address, bits))
	}
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("la lista no contiene ninguna IP ni CIDR")
	}
	return prefixes, nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.ListenAddr) == "" {
		return errors.New("config: listen_addr no puede estar vacío")
	}
	if strings.TrimSpace(c.DatabasePath) == "" {
		return errors.New("config: database_path no puede estar vacío")
	}
	if c.ScanInterval < 0 {
		return errors.New("config: scan_interval no puede ser negativo")
	}
	return nil
}
