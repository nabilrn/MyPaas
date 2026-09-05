#!/usr/bin/env bash

# Load a dotenv-style file as data. This intentionally never evals or sources
# the file itself, so values such as $(...), backticks, semicolons, or glob
# characters cannot become shell syntax.
load_env_file() {
  local env_file="$1"
  local raw line key value quote

  [[ -f "$env_file" ]] || {
    printf 'Missing environment file: %s\n' "$env_file" >&2
    return 1
  }

  while IFS= read -r raw || [[ -n "$raw" ]]; do
    line="$raw"
    # Trim CR from CRLF files. Do not otherwise mutate value whitespace.
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    [[ "$line" == *=* ]] || {
      printf 'Invalid environment line in %s: expected KEY=VALUE\n' "$env_file" >&2
      return 1
    }

    key="${line%%=*}"
    value="${line#*=}"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || {
      printf 'Invalid environment key in %s\n' "$env_file" >&2
      return 1
    }

    if (( ${#value} >= 2 )); then
      quote="${value:0:1}"
      if [[ ( "$quote" == "'" || "$quote" == '"' ) && "${value: -1}" == "$quote" ]]; then
        value="${value:1:${#value}-2}"
      fi
    fi

    export "$key=$value"
  done < "$env_file"
}
