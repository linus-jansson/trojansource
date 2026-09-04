# trojansource

`trojansource` scans Git-tracked source files for invisible Unicode formatting characters that can
be used in [Trojan Source](https://trojansource.codes/) attacks. It reports BiDi embedding,
override, and isolate controls, directional marks, and selected invisible characters. Visible
Unicode text, including Swedish text and emoji, remains allowed.

## Install and use

Run it without installation:

```sh
npx trojansource
```

Or install it in a project:

```sh
pnpm add -D trojansource
pnpm exec trojansource --all
```

The npm CLI dispatches to a precompiled Go binary for the current operating system and CPU
architecture. It supports Linux, macOS, and Windows on x64 and ARM64.

The default `--all` mode scans tracked and untracked, non-ignored files. Use `--staged` to scan
only staged additions, copies, modifications, and renames:

```sh
pnpm exec trojansource --staged
```

The command exits with status `1` when it finds an unsafe character and `2` for invalid arguments.

## Allowlisting files

To permit a file that intentionally contains one of these characters, add its repository-relative
path to `unicode-security-allowlist.json` in the repository root:

```json
{
  "files": ["fixtures/intentional-bidi.txt"]
}
```

Allowlisting skips the complete file. Prefer removing the character where possible.

## Go CLI and library

The repository also provides a native Go implementation with no external dependencies:

```sh
go install github.com/linus-jansson/trojansource/cmd/trojansource@latest
trojansource --staged
```

Use `github.com/linus-jansson/trojansource` as a Go library to call `ScanContent` or
`ScanRepository`. Its CLI has the same options, output, allowlist format, and exit codes as the
npm CLI. `ScanContent` returns findings without accessing the file system, and `ScanRepository`
accepts an `Options` value with `CWD`, `Mode`, and `AllowlistPath` fields.

## Publishing npm binaries

### Versioning

The root package and all six platform packages are a fixed
[Changesets](https://github.com/changesets/changesets) release group: they always receive the same
version. Add a changeset with `pnpm changeset` for every publishable change. Before creating a
GitHub release, run `pnpm version-packages` and commit the generated version updates and changelog.

### Publishing

Build the platform packages before publishing:

```sh
pnpm run build:npm-binaries
```

Publish all six `packages/*` packages first, then publish the root `trojansource` package. The
release workflow creates workspace-aware tarballs with pnpm, which replaces `workspace:*`
optional-dependency references with version `0.1.0`, then publishes the tarballs with npm. The
root package declares the platform packages as optional dependencies, so npm installs only the
matching native binary.

GitHub Actions runs this validation on pushes and pull requests. Publishing is triggered when a
GitHub release is published or manually from the **Publish npm packages** workflow. Configure the
`trojansource` package and every `@linus_janns/trojansource-*` platform package in npm to trust
the GitHub Actions publisher with these exact, case-sensitive values: owner
`linus-jansson`, repository `trojansource`, workflow filename `publish.yml`, and environment
`npm`. The workflow uses GitHub's OIDC identity; npm creates the provenance automatically, and no
`NPM_TOKEN` secret is required.
