#!/usr/bin/env bash

set -euo pipefail

usage() {
	cat <<'EOF'
Usage:
  scripts/generate-release-notes.sh <version> <from-ref> [--include-merges]

Examples:
  scripts/generate-release-notes.sh v1.2.2 v1.2.1
  scripts/generate-release-notes.sh v1.2.2 v1.2.1 --include-merges
EOF
}

require_cmd() {
	local name="$1"
	if ! command -v "${name}" >/dev/null 2>&1; then
		echo "Missing required command: ${name}" >&2
		exit 1
	fi
}

pluralize_files() {
	local count="$1"
	if [ "${count}" -eq 1 ]; then
		printf '%s' "file"
		return
	fi
	printf '%s' "files"
}

group_for_path() {
	local path="$1"
	IFS='/' read -r -a parts <<< "${path}"
	local count="${#parts[@]}"

	if [ "${count}" -eq 1 ]; then
		printf '%s' "${path}"
		return
	fi

	if [ "${parts[0]}" = "internal" ]; then
		if [ "${count}" -ge 4 ]; then
			printf '%s' "${parts[0]}/${parts[1]}/${parts[2]}"
			return
		fi
		printf '%s' "${parts[0]}/${parts[1]}"
		return
	fi

	if [ "${parts[0]}" = "cmd" ] || [ "${parts[0]}" = "scripts" ] || [ "${parts[0]}" = "MD-Files" ]; then
		printf '%s' "${parts[0]}"
		return
	fi

	if [ "${count}" -ge 2 ]; then
		printf '%s' "${parts[0]}/${parts[1]}"
		return
	fi

	printf '%s' "${path}"
}

sort_groups_by_count() {
	local -n counts_ref="$1"
	local group
	for group in "${!counts_ref[@]}"; do
		printf '%s\t%s\n' "${counts_ref[$group]}" "${group}"
	done | sort -rn | cut -f2
}

bullet_for_group() {
	local group="$1"
	case "${group}" in
		internal/cli)
			printf '%s\n' "Doprecyzowano warstwę CLI i orkiestracji \`toba create\`."
			;;
		internal/create)
			printf '%s\n' "Zmieniono executor i reguły działania dependency graph."
			;;
		cmd)
			printf '%s\n' "Zaktualizowano parser flag i wejście komend."
			;;
		README.md)
			printf '%s\n' "Odświeżono dokumentację użytkową."
			;;
		.github)
			printf '%s\n' "Zmieniono workflow publikacji w GitHub Actions."
			;;
		scripts)
			printf '%s\n' "Zautomatyzowano narzędzia release i publikacji."
			;;
		MD-Files)
			printf '%s\n' "Uzgodniono wewnętrzną dokumentację i checklisty."
			;;
		*)
			printf '%s\n' "Zmieniono obszar ${group}."
			;;
	esac
}

version="${1:-}"
from_ref="${2:-}"
include_merges="false"

if [ -z "${version}" ] || [ -z "${from_ref}" ]; then
	usage >&2
	exit 2
fi
shift 2 || true

while [ $# -gt 0 ]; do
	case "${1:-}" in
		--include-merges)
			include_merges="true"
			shift || true
			;;
		*)
			echo "Unknown argument: $1" >&2
			usage >&2
			exit 2
			;;
	esac
done

require_cmd git

if ! git rev-parse --verify "${from_ref}" >/dev/null 2>&1; then
	echo "Unknown git ref: ${from_ref}" >&2
	exit 1
fi

normalized_version="${version#v}"

log_args=(git log --reverse --format='%H%x09%s' "${from_ref}..HEAD")
if [ "${include_merges}" != "true" ]; then
	log_args=(git log --reverse --no-merges --format='%H%x09%s' "${from_ref}..HEAD")
fi

mapfile -t raw_commits < <("${log_args[@]}")
if [ "${#raw_commits[@]}" -eq 0 ]; then
	echo "No commits found in range ${from_ref}..HEAD" >&2
	exit 1
fi

mapfile -t raw_changes < <(git diff --name-status --find-renames "${from_ref}..HEAD")
if [ "${#raw_changes[@]}" -eq 0 ]; then
	echo "No changed files found in range ${from_ref}..HEAD" >&2
	exit 1
fi

declare -A group_counts=()
declare -A group_added=()
declare -A group_modified=()
declare -A group_deleted=()
declare -A group_renamed=()
changed_files=()

for raw in "${raw_changes[@]}"; do
	IFS=$'\t' read -r status path1 path2 <<< "${raw}"
	path="${path1}"
	case "${status}" in
		R*)
			path="${path2}"
			;;
	esac

	group="$(group_for_path "${path}")"
	changed_files+=("${path}")
	group_counts["${group}"]=$(( ${group_counts["${group}"]:-0} + 1 ))

	case "${status}" in
		A)
			group_added["${group}"]=$(( ${group_added["${group}"]:-0} + 1 ))
			;;
		M)
			group_modified["${group}"]=$(( ${group_modified["${group}"]:-0} + 1 ))
			;;
		D)
			group_deleted["${group}"]=$(( ${group_deleted["${group}"]:-0} + 1 ))
			;;
		R*)
			group_renamed["${group}"]=$(( ${group_renamed["${group}"]:-0} + 1 ))
			;;
		*)
			group_modified["${group}"]=$(( ${group_modified["${group}"]:-0} + 1 ))
			;;
	esac
done

mapfile -t ordered_groups < <(sort_groups_by_count group_counts)

summary_prefix="Wersja ${normalized_version}"
main_impact="Najważniejsza zmiana: release notes i changelog są teraz publikowane w spójnym, czytelnym formacie."

if [[ -n "${group_counts[internal/create]:-}" || -n "${group_counts[internal/cli]:-}" ]]; then
	summary_prefix="Wersja ${normalized_version} domyka zmiany wokół \`toba create\` i release"
	main_impact="Najważniejsza zmiana: \`toba create\` i zależny od niego workflow mogą być teraz opisywane i publikowane w bardziej przewidywalnym formacie."
fi

if [[ -n "${group_counts[.github]:-}" || -n "${group_counts[scripts]:-}" ]]; then
	main_impact="Najważniejsza zmiana: release workflow generuje i publikuje opis wydania automatycznie."
fi

cat <<EOF
## ToBA ${normalized_version}

${summary_prefix}. Ten release zbiera zmiany z zakresu ${from_ref}..HEAD i publikuje je w jednolitym układzie.

${main_impact}

### Najważniejsze zmiany
EOF

index=0
for group in "${ordered_groups[@]}"; do
	printf -- '- %s\n' "$(bullet_for_group "${group}")"
	index=$((index + 1))
	if [ "${index}" -eq 5 ]; then
		break
	fi
done

cat <<'EOF'

## Changelog
EOF

for raw in "${raw_commits[@]}"; do
	IFS=$'\t' read -r sha subject <<< "${raw}"
	printf -- '* %s %s\n' "${sha}" "${subject}"
done
