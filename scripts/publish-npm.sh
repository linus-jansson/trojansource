#!/usr/bin/env bash
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="$(node -p "require('./package.json').version")"
platform_packages=(packages/*)

for package in "${platform_packages[@]}"; do
	package_version="$(node -p "require('./$package/package.json').version")"
	if [[ "$package_version" != "$version" ]]; then
		echo "$package/package.json has version $package_version; expected $version." >&2
		exit 1
	fi
done

pnpm test
pnpm run build:npm-binaries

temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT
mkdir -p "$temporary_directory/platforms" "$temporary_directory/launcher"

for package in "${platform_packages[@]}"; do
	pnpm --dir "$package" pack --pack-destination "$temporary_directory/platforms" >/dev/null
done
pnpm pack --pack-destination "$temporary_directory/launcher" >/dev/null

read -r -s -p "Npm one-time password: " npm_otp
printf '\n'

for package in "$temporary_directory"/platforms/*.tgz; do
	npm publish "$package" --access public --otp="$npm_otp"
done
npm publish "$temporary_directory"/launcher/*.tgz --otp="$npm_otp"
