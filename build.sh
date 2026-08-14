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
  test     Test every project-owned command, internal package, and integration test.
  check    Run gofmt verification, go vet, and the complete project test suite.
  man      List source man pages packaged with releases.
  man-build Create gzip-compressed man pages in dist/man.
  man-install Install compressed man pages into prefix/share/man/man1.
  list     List every command that build and install operate on.
  install  Build every command and install it into prefix/bin.
  clean    Remove generated build output from dist/.
  help     Show this help text.

Options:
  prefix=<path>  Installation prefix (default: /usr/local).

Examples:
  ./build.sh build
  ./build.sh check
  ./build.sh list
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

list_commands() {
	for command_dir in cmd/*; do
		[ -d "$command_dir" ] || continue
		basename "$command_dir"
	done
}

list_man_pages() {
	find man -type f -name '*.[1-9]' -print | sort
}

build_man_pages() {
	mkdir -p "$bin_dir/man"
	rm -f "$bin_dir/man"/*.1.gz
	for page in man/*.1; do
		[ -s "$page" ] || continue
		gzip -9 -c "$page" > "$bin_dir/man/$(basename "$page").gz"
	done
}

install_man_pages() {
	build_man_pages
	mkdir -p "$prefix/share/man/man1"
	for page in "$bin_dir/man"/*.1.gz; do
		[ -f "$page" ] || continue
		install -m 0644 "$page" "$prefix/share/man/man1/$(basename "$page")"
	done
}

test_project() {
	"$go_cmd" test ./cmd/... ./internal/... ./tests
}

check_project() {
	unformatted=$(gofmt -l cmd internal tests)
	if [ -n "$unformatted" ]; then
		echo "build.sh: gofmt required for:" >&2
		echo "$unformatted" >&2
		return 1
	fi
	"$go_cmd" vet ./cmd/... ./internal/... ./tests
	test_project
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
	build|test|check|man|man-build|man-install|list|install|clean|help|-h|--help)
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
	test)
		test_project
		;;
	check)
		check_project
		;;
	man)
		list_man_pages
		;;
	man-build)
		build_man_pages
		;;
	man-install)
		install_man_pages
		;;
	list)
		list_commands
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
