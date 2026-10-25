#!/usr/bin/env bash
# Aplica la última versión de Cinecat: trae los cambios, reconstruye las
# imágenes y reinicia los contenedores sin perder datos (la base de datos y
# los certificados de Caddy viven en volúmenes aparte).
set -euo pipefail
cd "$(dirname "$0")/.."

git pull --ff-only
docker compose build
docker compose up -d
docker compose ps
