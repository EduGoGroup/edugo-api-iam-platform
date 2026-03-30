#!/usr/bin/env bash

set -e

# Asegurarnos de que el script se pueda correr desde cualquier sitio
# cambiando el cd al dir raíz del proyecto
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/.."

# ==============================================================================
# Verificaciones Previas
# ==============================================================================

# 1. Verificar gh CLI
if ! command -v gh &> /dev/null; then
    echo "❌ ERROR: gh (GitHub CLI) no está instalado. Por favor instálalo y autentícate."
    exit 1
fi

# 2. Verificar estatus de autenticación
if ! gh auth status &> /dev/null; then
    echo "❌ ERROR: No estás autenticado en GitHub CLI. Corre 'gh auth login' y reintenta."
    exit 1
fi

# ==============================================================================
# Información de versión
# ==============================================================================

echo "🔍 Consultando la última versión lanzada desde main..."
git fetch --tags -q 2>/dev/null || true
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "Ninguna")

echo "========================================="
echo "📦 Última versión detectada: $LATEST_TAG"
echo "========================================="
echo ""

# Pedir la nueva versión
read -p "👉 Introduce el número de la NUEVA versión (sin la 'v', ej. 0.1.0): " NEW_VERSION
if [ -z "$NEW_VERSION" ]; then
    echo "❌ Error: La versión no puede estar vacía."
    exit 1
fi

# Validar formato semver (muy simple)
if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "❌ Error: El formato debe ser estrictamente x.y.z (ejemplo: 1.0.2)"
    exit 1
fi

echo ""
echo "========================================="
echo "⚙️  Selecciona el tipo de actualización:"
echo "1) patch  - Correcciones de errores (ej. 0.1.0 -> 0.1.1)"
echo "2) minor  - Nuevas características retrocompatibles (ej. 0.1.0 -> 0.2.0)"
echo "3) major  - Cambios no retrocompatibles / producción (ej. 0.1.0 -> 1.0.0)"
echo "========================================="
read -p "Opcion [1-3] (por defecto 1 - patch): " BUMP_OPT

BUMP_TYPE="patch"
case $BUMP_OPT in
    2) BUMP_TYPE="minor" ;;
    3) BUMP_TYPE="major" ;;
esac

echo ""
echo "🚀 Iniciando proceso para v$NEW_VERSION ($BUMP_TYPE)..."
echo "──────────────────────────────────────────────────"

# ==============================================================================
# 1. Pipeline: Manual Release
# ==============================================================================

echo "⏳ Despachando github action: manual-release.yml a la rama 'main'..."
gh workflow run manual-release.yml -f version="$NEW_VERSION" -f bump_type="$BUMP_TYPE" -f enable_auto_release="yes" --ref main

# Pequeña pausa para que GH asigne el ID al workflow dispatch
echo "⏳ Esperando 5 segundos a que el workflow empiece..."
sleep 5

# Obtener ID del run de manual-release (el más reciente disparado hace poco en la rama main)
MR_RUN_ID=$(gh run list --workflow=manual-release.yml -b main --limit 1 --json databaseId -q ".[0].databaseId")

if [ -z "$MR_RUN_ID" ]; then
    echo "❌ No se pudo determinar el Run ID del action 'manual-release.yml'."
    echo "Es posible que haya un problema o no se despachó exitosamente."
    exit 1
fi

echo "👀 Monitoreando pipeline de Release (Run ID: $MR_RUN_ID). Esto puede tomar unos minutos..."
# gh run watch fallará (exit > 0) si el pipeline termina en error
gh run watch "$MR_RUN_ID" --exit-status

if [ $? -ne 0 ]; then
    echo "❌ ERROR: El pipeline de release (manual-release.yml) falló. Abortando el despliegue a Azure."
    exit 1
fi

echo "✅ Release v$NEW_VERSION generado con éxito."
echo ""

# ==============================================================================
# 2. Pipeline: Deploy to Azure
# ==============================================================================

echo "🚀 Despachando github action: deploy-to-azure.yml a la rama 'main'..."
gh workflow run deploy-to-azure.yml -f image_tag="v$NEW_VERSION" --ref main

echo "⏳ Esperando 5 segundos a que el workflow empiece..."
sleep 5

DA_RUN_ID=$(gh run list --workflow=deploy-to-azure.yml -b main --limit 1 --json databaseId -q ".[0].databaseId")

if [ -z "$DA_RUN_ID" ]; then
    echo "❌ No se pudo determinar el Run ID del action 'deploy-to-azure.yml'."
    exit 1
fi

echo "👀 Monitoreando pipeline de Despliegue en Azure (Run ID: $DA_RUN_ID). Esto también tomará un tiempo..."
gh run watch "$DA_RUN_ID" --exit-status

if [ $? -ne 0 ]; then
    echo "❌ ERROR: El pipeline de despliegue a Azure (deploy-to-azure.yml) falló."
    echo "Podrás ver más detalles entrando a los Actions de repositorio en GitHub."
    exit 1
fi

echo "──────────────────────────────────────────────────"
echo "🎉 ¡Flujo Completo Completado de Forma Exitosa!"
echo "   - Creado release: v$NEW_VERSION"
echo "   - Desplegada imagen en Azure con el nuevo release."
echo "──────────────────────────────────────────────────"
