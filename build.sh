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
Usage: ./build.sh [build] [install] [clean] [prefix=<path>]

Targets:
  build    Build jwalk, envsub, and hashsum into dist/ (default).
  install  Build the commands and install them into prefix/bin.
  clean    Remove generated build output from dist/.

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
	"$go_cmd" build -trimpath -o "$bin_dir/jwalk" ./cmd/jwalk
	"$go_cmd" build -trimpath -o "$bin_dir/envsub" ./cmd/envsub
	"$go_cmd" build -trimpath -o "$bin_dir/hashsum" ./cmd/hashsum
}

install_binaries() {
	build
	mkdir -p "$prefix/bin"
	for binary in jwalk envsub hashsum; do
		install -m 0755 "$bin_dir/$binary" "$prefix/bin/$binary"
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
	build|install|clean|-h|--help|help)
		;;
	*)
		echo "build.sh: unknown target or option: $argument" >&2
		usage >&2
		exit 2
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
	-h|--help|help)
		usage
		;;
	prefix=*)
		;;
	esac
done
