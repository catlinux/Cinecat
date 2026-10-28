package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	signingKeyFile    = "signing.key"
	signingKeyLength  = 32
	signingKeyOctal   = 0o600
	signingKeyDirMask = 0o700
)

// loadSigningKey lee o crea la clave HMAC junto a la base de datos. La clave
// sobrevive a los reinicios y a las actualizaciones; con una base en memoria
// se genera una aleatoria que solo dura lo que el proceso.
func loadSigningKey(databasePath string) ([]byte, error) {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return newSigningKey()
	}

	dir := filepath.Dir(path)
	keyPath := filepath.Join(dir, signingKeyFile)
	data, err := os.ReadFile(keyPath)
	if err == nil {
		decoded, decodeErr := hex.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr == nil && len(decoded) == signingKeyLength {
			return decoded, nil
		}
		log.Printf("cinecatd: %s no es una clave válida; se regenera", keyPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("cinecatd: no se pudo leer %s: %v; se regenera", keyPath, err)
	}

	key, err := newSigningKey()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, signingKeyDirMask); err != nil {
		return nil, fmt.Errorf("cinecatd: crear el directorio de la clave: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".signing.key-*")
	if err != nil {
		return nil, fmt.Errorf("cinecatd: crear la clave temporal: %w", err)
	}
	tempPath := temp.Name()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(signingKeyOctal); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("cinecatd: permisos de la clave: %w", err)
	}
	if _, err := temp.WriteString(hex.EncodeToString(key)); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("cinecatd: escribir la clave: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("cinecatd: sincronizar la clave: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("cinecatd: cerrar la clave: %w", err)
	}
	if err := os.Rename(tempPath, keyPath); err != nil {
		return nil, fmt.Errorf("cinecatd: guardar la clave: %w", err)
	}
	complete = true
	return key, nil
}

func newSigningKey() ([]byte, error) {
	key := make([]byte, signingKeyLength)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("cinecatd: generar la clave de firma: %w", err)
	}
	return key, nil
}
