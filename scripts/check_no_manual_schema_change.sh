#!/usr/bin/env bash
# Local pre-commit guard: schema changes to a live database (dev/prod) MUST go
# through an Atlas migration file. This hook blocks commits whose staged diff
# would (a) edit or delete an existing migration file, or (b) sneak schema
# DDL through anything OTHER than backend/migrations/ (e.g. psql commands
# embedded in shell, ad-hoc scripts, README examples that look runnable).
#
# Adding a new migration file in backend/migrations/ is fine — that IS the
# intended path. Editing an already-applied migration is not (would change
# atlas.sum and silently re-apply on the next env).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT}"

# Only look at the staged diff for this commit.
# `git diff --staged --name-status`:
#   M <file>  modified
#   A <file>  added
#   D <file>  deleted
#   R <old> -> <new>  renamed
STAGED=()
while IFS= read -r line; do STAGED+=("$line"); done < <(git diff --staged --name-status --diff-filter=MARD || true)

bad=0

for entry in "${STAGED[@]}"; do
  status="${entry%%	*}"
  path="${entry##*	}"
  # Renames are "old\tnew"; collapse to the new path.
  if [[ "$path" == *"	"* ]]; then path="${path##*	}"; fi

  # (a) Don't allow editing/deleting/renaming files inside backend/migrations/
  # other than the new-file (A) case. Hand-editing an applied migration is
  # the most common silent break (changes DDL without updating atlas.sum).
  if [[ "$path" == backend/migrations/* ]] && \
     [[ "$status" == "M" || "$status" == "D" || "$status" == "R" ]]; then
    echo "check_no_manual_schema_change: '$path' ($status) modifies an already-applied migration." >&2
    echo "  → Add a NEW migration instead: backend/migrations/\$(date +%Y%m%d%H%M%S)_<name>.sql" >&2
    echo "  → And regenerate atlas.sum: cd backend && make db-migrate-hash" >&2
    bad=1
  fi

  # (b) Anything outside backend/migrations/ that looks like a runnable DDL
  # statement (ALTER/CREATE/DROP against a live object) — block it. Whitelist
  # the migrations dir, README/docs, archive/, and migrations' SQL itself.
  case "$path" in
    backend/migrations/*|backend/migrations/archive/*) continue ;;
    *.md|*.txt) continue ;;
    scripts/check_migration_*.sh) continue ;;  # this script + sibling
  esac
  if grep -nqE '\b(ALTER\s+TABLE|CREATE\s+TABLE|CREATE\s+INDEX|CREATE\s+OR\s+REPLACE\s+FUNCTION|CREATE\s+TRIGGER|DROP\s+TABLE|DROP\s+INDEX|DROP\s+CONSTRAINT|TRUNCATE)\b' "$path" 2>/dev/null; then
    if [[ "$status" != "D" ]]; then
      echo "check_no_manual_schema_change: '$path' ($status) contains runnable DDL but is NOT under backend/migrations/." >&2
      echo "  → Schema changes only land via a new migration file under backend/migrations/" >&2
      echo "  → If this is a docs example / commented-out block, prefix the line with '-- ' (SQL) or '#' (shell) so it doesn't match." >&2
      bad=1
    fi
  fi
done

if [[ "$bad" -ne 0 ]]; then
  cat >&2 <<'EOF'

Schema changes to dev/prod MUST go through Atlas migrations. See
docs/agents/AI-RULES.md ("Schema changes via Atlas migrations (mandatory)").
EOF
  exit 1
fi
echo "check_no_manual_schema_change: OK"
