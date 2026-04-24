# How to release a new version

**IMPORTANT:** Do not push release tags manually. The entire release process is automated via GitHub Actions.

## Plugin Validator

Use the ["Bump Version and release"](https://github.com/grafana/plugin-validator/actions/workflows/do-release.yml) workflow:

1. Click **"Run workflow"**
2. Keep branch: `main`
3. Select the semver type: `patch`, `minor`, or `major`
4. Click **"Run workflow"**

### Which semver type to choose?

- **patch** -- bug fixes, dependency updates, documentation changes
- **minor** -- new features (e.g. adding a new validator)
- **major** -- breaking changes to the CLI interface, output format, or public API

### What happens after the workflow runs?

The workflow bumps the version in `package.json`, commits to `main`, and pushes a `v*` tag. The tag push automatically triggers the ["Create release and publish"](https://github.com/grafana/plugin-validator/actions/workflows/release.yml) workflow, which:

1. Builds binaries via GoReleaser (Linux, Windows, Darwin) and creates a **GitHub Release**
2. Publishes to **npm** (`@grafana/plugin-validator`)
3. Pushes a **Docker image** to `grafana/plugin-validator-cli` (tagged with the version and `latest`)

## MCP Server

The MCP server has its own, independent release track. Use the ["Release MCP Server"](https://github.com/grafana/plugin-validator/actions/workflows/do-release-mcp.yml) workflow:

1. Click **"Run workflow"**
2. Keep branch: `main`
3. Select the semver type: `patch`, `minor`, or `major`
4. Click **"Run workflow"**

This bumps the version in `mcp-package/package.json`, commits to `main`, and pushes a `mcp/v*` tag. The tag push automatically triggers the ["Create MCP release and publish"](https://github.com/grafana/plugin-validator/actions/workflows/release-mcp.yml) workflow, which:

1. Builds binaries via GoReleaser (Linux, Windows, Darwin -- both amd64 and arm64) and creates a **GitHub Release**
2. Publishes to **npm** (`@grafana/plugin-validator-mcp`)

## Permissions

> [!NOTE]
> For security purposes, external contributors don't usually have permissions to create new releases.

## GitHub or NPM tokens expiration

GitHub and NPM tokens are required to auto-publish. These tokens eventually expire. If there's an error pushing tags to GitHub or updates to npm, review the token expiration. See the [secrets usage table](https://grafana.github.io/grafana-catalog-team/secrets-rotation/secrets-usage-table/) for details on which secrets are used and who owns them.
