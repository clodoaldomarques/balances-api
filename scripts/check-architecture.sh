#!/usr/bin/env bash
#
# Guarda arquitetural da spec 005-hexagonal-adapters.
#
# Falha quando:
#   - internal/core/** importa adapters, infra legada, cloud SDK, Echo, config ou SQL;
#   - construtores de adapters/caso de uso sao chamados fora da raiz de composicao (cmd/)
#     e dos proprios adapters.
#
# Uso: make arch-check

set -uo pipefail

cd "$(dirname "$0")/.."

violations=0

report() {
	printf 'arch-check: %s\n' "$1" >&2
	violations=1
}

check_import() {
	local label="$1"
	local pattern="$2"
	local path="$3"
	local match

	while IFS= read -r match; do
		[ -z "$match" ] && continue
		report "$label: $match"
	done < <(grep -R -n -E --include='*.go' "$pattern" "$path" 2>/dev/null || true)
}

check_import 'core importa adapters' '"[^"]*/internal/adapters[^"]*"' internal/core
check_import 'core importa infra legada' '"[^"]*/internal/(infra|domain)[^"]*"' internal/core
check_import 'core importa cloud SDK' '("github.com/aws/[^"]*"|"[^"]*core-sdk/pkg/aws[^"]*")' internal/core
check_import 'core importa Echo' '"github.com/labstack/echo/v4[^"]*"' internal/core
check_import 'core importa config' '"github.com/clodoaldomarques/balances-api/config"' internal/core
check_import 'core importa SQL/driver' '("database/sql"|"github.com/go-sql-driver/mysql")' internal/core

while IFS= read -r file; do
	while IFS= read -r match; do
		[ -z "$match" ] && continue
		report "composicao fora de cmd/: $match"
	done < <(grep -n -E \
		'(config\.New\(|application\.NewService\(|mysql\.Connect\(|mysql\.NewRepository\(|sns\.NewPublisher\(|snsadapter\.NewPublisher\(|server\.New\(|accounts\.New\()' \
		"$file" 2>/dev/null || true)
done < <(find . -type f -name '*.go' \
	-not -path './cmd/*' \
	-not -path './config/*' \
	-not -path './internal/adapters/*' \
	-not -path './.git/*')

if [ "$violations" -ne 0 ]; then
	exit 1
fi

printf 'arch-check: ok\n'
