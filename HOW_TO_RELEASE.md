# How to release a new version

**IMPORTANT:** Releases are driven by [release-please](https://github.com/googleapis/release-please-action) on top of [conventional commits](https://www.conventionalcommits.org/). You don't bump versions or push tags manually.

## How it works

The [Release Please](https://github.com/grafana/plugin-validator/actions/workflows/release-please.yml) workflow runs on every push to `main`. It looks at the conventional commits since the last release and maintains an open release PR per package (one for the validator, one for the MCP server).

When you merge a release PR:

1. release-please creates the version tag and a GitHub Release with the changelog body.
2. The tag push triggers the corresponding publish workflow.

| Package | Tag format | Triggered workflow | Publishes to |
| --- | --- | --- | --- |
| Plugin Validator (root) | `plugin-validator/v*` | [Create release and publish](https://github.com/grafana/plugin-validator/actions/workflows/release.yml) | GitHub Release assets (GoReleaser), npm `@grafana/plugin-validator`, Docker `grafana/plugin-validator-cli` |
| MCP Server | `mcp/v*` | [Create MCP release and publish](https://github.com/grafana/plugin-validator/actions/workflows/release-mcp.yml) | GitHub Release assets (GoReleaser), npm `@grafana/plugin-validator-mcp` |

GoReleaser runs after release-please with the default `release.mode: keep-existing`, so it preserves the changelog body release-please put on the release and only attaches the binary assets.

## Cutting a release

1. Make sure your changes have landed on `main` with conventional commit messages (`feat:`, `fix:`, `chore:`, etc.). The bump type follows the commit types: `feat` → minor, `fix` → patch, `feat!` / `BREAKING CHANGE` → major.
2. Open the release PR for the package you want to ship (titled `chore(plugin-validator): release X.Y.Z` or `chore(mcp): release X.Y.Z`). Review the proposed changelog and version bump.
3. Squash-merge the release PR. The publish workflow runs automatically.

If no release PR exists, the **Release Please** workflow can also be triggered manually from the Actions tab — it'll open one based on the current state of `main`.

## Conventional commit cheat sheet

Use these prefixes; release-please groups them in the changelog:

- `feat:` -- new feature (minor bump)
- `fix:` -- bug fix (patch bump)
- `feat!:` / footer `BREAKING CHANGE:` -- breaking change (major bump)
- `docs:`, `style:`, `refactor:`, `perf:`, `test:`, `build:`, `ci:`, `chore:`, `revert:` -- recorded in the changelog without changing the version (unless paired with `!` or `BREAKING CHANGE`)

For MCP-server-only changes, scope your commit (e.g. `feat(mcp): add new tool`) and modify files under `mcp-package/` -- release-please uses paths to decide which package's version to bump.

## Permissions

> [!NOTE]
> External contributors don't usually have permission to merge release PRs. Ask a maintainer.

## Token expirations

The publish workflows use the `plugins-platform-bot-app` GitHub App and an npm token; both can expire. If a publish step fails with an auth error, check the [secrets usage table](https://grafana.github.io/grafana-catalog-team/secrets-rotation/secrets-usage-table/).
