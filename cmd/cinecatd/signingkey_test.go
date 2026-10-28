package main

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSigningKeyCreaYReutiliza(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "cinecat.db")

	first, err := loadSigningKey(databasePath)
	if err != nil {
		t.Fatalf("crear clave: %v", err)
	}
	if len(first) != signingKeyLength {
		t.Fatalf("longitud = %d", len(first))
	}
	keyPath := filepath.Join(filepath.Dir(databasePath), signingKeyFile)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("leer clave: %v", err)
	}
	decoded, err := hex.DecodeString(string(data))
	if err != nil || len(decoded) != signingKeyLength {
		t.Fatalf("clave guardada inválida: %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatalf("stat clave: %v", err)
		}
		if got := info.Mode().Perm(); got != signingKeyOctal {
			t.Fatalf("permisos = %o, se esperaba %o", got, signingKeyOctal)
		}
	}

	second, err := loadSigningKey(databasePath)
	if err != nil {
		t.Fatalf("reutilizar clave: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("la clave debe persistir entre lecturas")
	}
}

func TestSigningKeyRegeneraSiEstaCorrupta(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "cinecat.db")
	keyPath := filepath.Join(dir, signingKeyFile)
	if err := os.WriteFile(keyPath, []byte("no es hexadecimal"), 0o644); err != nil {
		t.Fatalf("preparar clave corrupta: %v", err)
	}

	key, err := loadSigningKey(databasePath)
	if err != nil {
		t.Fatalf("regenerar clave: %v", err)
	}
	if len(key) != signingKeyLength {
		t.Fatalf("longitud = %d", len(key))
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("leer clave regenerada: %v", err)
	}
	decoded, err := hex.DecodeString(string(data))
	if err != nil || len(decoded) != signingKeyLength {
		t.Fatalf("clave regenerada inválida: %q, %v", data, err)
	}
}

func TestSigningKeyRegeneraSiLaLongitudEsIncorrecta(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "cinecat.db")
	keyPath := filepath.Join(dir, signingKeyFile)
	if err := os.WriteFile(keyPath, []byte("00112233445566778899aabbccddeeff"), 0o600); err != nil {
		t.Fatalf("preparar clave corta: %v", err)
	}

	key, err := loadSigningKey(databasePath)
	if err != nil {
		t.Fatalf("regenerar clave: %v", err)
	}
	if len(key) != signingKeyLength {
		t.Fatalf("longitud = %d", len(key))
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("leer clave regenerada: %v", err)
	}
	decoded, err := hex.DecodeString(string(data))
	if err != nil || len(decoded) != signingKeyLength {
		t.Fatalf("clave regenerada inválida: %q, %v", data, err)
	}
}

func TestSigningKeyEnMemoria(t *testing.T) {
	first, err := loadSigningKey(":memory:")
	if err != nil {
		t.Fatalf("clave en memoria: %v", err)
	}
	second, err := loadSigningKey("")
	if err != nil {
		t.Fatalf("clave vacía: %v", err)
	}
	if len(first) != signingKeyLength || len(second) != signingKeyLength {
		t.Fatalf("longitudes = %d / %d", len(first), len(second))
	}
	if string(first) == string(second) {
		t.Fatal("las claves en memoria deben ser aleatorias")
	}
}
