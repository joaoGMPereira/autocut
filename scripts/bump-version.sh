#!/usr/bin/env bash
# scripts/bump-version.sh — Incrementa a versão do AutoCut
# Atualiza apps/web/package.json, apps/desktop/package.json e packages/shared/package.json em sincronia.
#
# Uso:
#   ./scripts/bump-version.sh          (interativo)
#   ./scripts/bump-version.sh patch    (direto)
#   ./scripts/bump-version.sh minor
#   ./scripts/bump-version.sh major

set -e

DESKTOP_PKG="apps/desktop/package.json"
WEB_PKG="apps/web/package.json"
SHARED_PKG="packages/shared/package.json"

# Ler versão atual do desktop (fonte da verdade)
CURRENT=$(node -p "require('./$DESKTOP_PKG').version")

# Separar em major.minor.patch
IFS='.' read -r V_MAJOR V_MINOR V_PATCH <<< "$CURRENT"

# Calcular candidatos
NEXT_PATCH="$V_MAJOR.$V_MINOR.$((V_PATCH + 1))"
NEXT_MINOR="$V_MAJOR.$((V_MINOR + 1)).0"
NEXT_MAJOR="$((V_MAJOR + 1)).0.0"

echo ""
echo "  AutoCut — Bump de Versão"
echo "  Versão atual: $CURRENT"
echo ""

# Determinar tipo pelo argumento ou perguntar interativamente
BUMP_TYPE="${1:-}"

if [ -z "$BUMP_TYPE" ]; then
  echo "  Qual componente incrementar?"
  echo ""
  echo "    1) patch  →  $NEXT_PATCH  (bug fix, ajuste pequeno)"
  echo "    2) minor  →  $NEXT_MINOR  (nova funcionalidade, reset do patch)"
  echo "    3) major  →  $NEXT_MAJOR  (breaking change, reset de minor e patch)"
  echo ""
  read -rp "  Escolha [1/2/3]: " CHOICE
  echo ""

  case "$CHOICE" in
    1|patch) BUMP_TYPE="patch" ;;
    2|minor) BUMP_TYPE="minor" ;;
    3|major) BUMP_TYPE="major" ;;
    *)
      echo "  Opção inválida: '$CHOICE'. Use 1, 2 ou 3."
      exit 1
      ;;
  esac
fi

# Calcular nova versão
case "$BUMP_TYPE" in
  patch) NEW_VERSION="$NEXT_PATCH" ;;
  minor) NEW_VERSION="$NEXT_MINOR" ;;
  major) NEW_VERSION="$NEXT_MAJOR" ;;
  *)
    echo "  Tipo inválido: '$BUMP_TYPE'. Use patch, minor ou major."
    exit 1
    ;;
esac

echo "  $CURRENT  →  $NEW_VERSION"
echo ""

# Atualizar apps/web/package.json
node -e "
  const fs = require('fs');
  const path = '$WEB_PKG';
  const pkg = JSON.parse(fs.readFileSync(path, 'utf8'));
  pkg.version = '$NEW_VERSION';
  fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + '\n');
"

# Atualizar apps/desktop/package.json
node -e "
  const fs = require('fs');
  const path = '$DESKTOP_PKG';
  const pkg = JSON.parse(fs.readFileSync(path, 'utf8'));
  pkg.version = '$NEW_VERSION';
  fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + '\n');
"

# Atualizar packages/shared/package.json
node -e "
  const fs = require('fs');
  const path = '$SHARED_PKG';
  const pkg = JSON.parse(fs.readFileSync(path, 'utf8'));
  pkg.version = '$NEW_VERSION';
  fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + '\n');
"

echo "  Atualizado:"
echo "    $WEB_PKG        → $NEW_VERSION"
echo "    $DESKTOP_PKG → $NEW_VERSION"
echo "    $SHARED_PKG  → $NEW_VERSION"
echo ""
