#!/bin/sh
# Build helper for the Missing Utils command-line tools.

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

go_cmd=${GO:-go}
bin_dir=dist
prefix=/usr/local

usage() {
	cat <<'EOF'
Usage: ./build.sh [target] [prefix=<path>]

Targets:
  build    Build every command in cmd/ into dist/ (default).
  install  Build every command and install it into prefix/bin.
  clean    Remove generated build output from dist/.
  help     Show this help text.

Options:
  prefix=<path>  Installation prefix (default: /usr/local).

Examples:
  ./build.sh build
  ./build.sh install prefix="$HOME/.local"
  ./build.sh clean
EOF
}

build() {
	mkdir -p "$bin_dir"
	for command_dir in cmd/*; do
		[ -d "$command_dir" ] || continue
		command=${command_dir#cmd/}
		"$go_cmd" build -trimpath -o "$bin_dir/$command" "./$command_dir"
	done
}

install_binaries() {
	build
	mkdir -p "$prefix/bin"
	for binary in "$bin_dir"/*; do
		[ -f "$binary" ] || continue
		install -m 0755 "$binary" "$prefix/bin/$(basename "$binary")"
	done
}

clean() {
	rm -rf "$bin_dir"
}

for argument in "$@"; do
	case "$argument" in
	prefix=*)
		prefix=${argument#prefix=}
		if [ -z "$prefix" ]; then
			echo "build.sh: prefix must not be empty" >&2
			exit 2
		fi
		;;
	build|install|clean|help|-h|--help)
		;;
	*)
		echo "build.sh: unknown target or option: $argument" >&2
		usage >&2
		exit 2
		;;
	esac
done

for argument in "$@"; do
	case "$argument" in
	help|-h|--help)
		usage
		exit 0
		;;
	esac
done

if [ "$#" -eq 0 ]; then
	build
fi

for argument in "$@"; do
	case "$argument" in
	build)
		build
		;;
	install)
		install_binaries
		;;
	clean)
		clean
		;;
	prefix=*)
		;;
	esac
done
