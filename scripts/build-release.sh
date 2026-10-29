#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo 'Usage: bash scripts/build-release.sh VERSION GOOS GOARCH OUTPUT_DIR' >&2
  exit 2
fi
release_version="$1"
release_goos="$2"
release_goarch="$3"
release_output_dir="$4"

if [[ ! "$release_version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  echo 'VERSION must be a semantic version tag such as v0.1.0 or v0.1.0-rc.1' >&2
  exit 2
fi
case "$release_goos/$release_goarch" in
  linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64|windows/arm64) ;;
  *) echo "Unsupported release target: $release_goos/$release_goarch" >&2; exit 2 ;;
esac

release_repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$release_repo_dir"
mkdir -p -- "$release_output_dir"
release_output_dir="$(cd -- "$release_output_dir" && pwd)"
release_workdir="$(mktemp -d)"
trap 'rm -rf -- "$release_workdir"' EXIT
release_binary=hexgogen
if [[ "$release_goos" == windows ]]; then
  release_binary=hexgogen.exe
fi

CGO_ENABLED=0 GOOS="$release_goos" GOARCH="$release_goarch" \
  go build -trimpath \
  -ldflags "-s -w -X github.com/aoamusat/hexgogen/internal/cli.Version=$release_version" \
  -o "$release_workdir/$release_binary" ./cmd/hexgogen

cp README.md "$release_workdir/README.md"
release_files=("$release_binary" README.md)
for release_license in LICENSE LICENSE.txt LICENSE.md; do
  if [[ -f "$release_license" ]]; then
    cp "$release_license" "$release_workdir/$release_license"
    release_files+=("$release_license")
  fi
done

release_name="hexgogen_${release_version}_${release_goos}_${release_goarch}"
if [[ "$release_goos" == windows ]]; then
  release_archive="$release_name.zip"
  (cd "$release_workdir" && zip -q "$release_workdir/$release_archive" "${release_files[@]}")
else
  release_archive="$release_name.tar.gz"
  tar -C "$release_workdir" -czf "$release_workdir/$release_archive" "${release_files[@]}"
fi
mv -- "$release_workdir/$release_archive" "$release_output_dir/$release_archive"
(cd "$release_output_dir" && sha256sum "$release_archive" > "$release_archive.sha256")
printf 'Built %s\n' "$release_output_dir/$release_archive"
