# CanvasProbe

CanvasProbe is a small command-line client for inspecting design data through the Figma REST API. Read commands return filterable JSON; explicit comment commands provide a narrow write-back path for review workflows.

CanvasProbe is an independent, unofficial tool compatible with the Figma REST API. It is not affiliated with, sponsored by, or endorsed by Figma, Inc. Figma is a trademark of Figma, Inc.

## Install

CanvasProbe requires Go 1.25 or newer.

```sh
go install github.com/dsuranov/canvasprobe@latest
```

Or build the current checkout:

```sh
go build -trimpath -o ./canvasprobe .
```

## Authentication

Create a scoped token in the Figma developer settings and expose it through one of these sources, in priority order:

1. `--token-stdin`
2. `CANVASPROBE_TOKEN`
3. `FIGMA_API_TOKEN`
4. `token` in `~/.config/canvasprobe/config.yaml`

Tokens are never accepted as command-line values. Command-line arguments are commonly visible to other processes and shell history.

```sh
export CANVASPROBE_TOKEN='figd_...'
canvasprobe me

printf '%s' "$CANVASPROBE_TOKEN" | canvasprobe --token-stdin me
```

Example config, which must have mode `0600`:

```yaml
token: figd_...
default_format: json
cache_ttl: 300
timeout: 30
```

See [authentication and scopes](docs/authentication.md) for PAT, OAuth, and plan-token limitations.

## Commands

```text
canvasprobe me
canvasprobe file <key> [--depth N] [--ids 1:2,3:4] [--version ID] [--jq EXPR]
canvasprobe file <key> nodes <ids> [--depth N] [--jq EXPR]
canvasprobe file <key> components [--fields name,key] [--jq EXPR]
canvasprobe file <key> styles [--fields name,key] [--jq EXPR]

canvasprobe comments list <file-key>
canvasprobe comments create <file-key> --message "Ready for review"
canvasprobe comments create <file-key> --message "Reply" --reply-to <comment-id>
canvasprobe comments delete <file-key> <comment-id> --yes

canvasprobe cache dir
canvasprobe cache status
canvasprobe cache purge
```

`file` defaults to server-side `--depth 2`. Pass `--depth 0` to request the complete file. CanvasProbe never applies a second generic depth truncation to the response envelope.

`--fields` is intentionally limited to the flat component/style lists. Use `--jq` for file trees and node envelopes:

```sh
canvasprobe file abc123 --depth 3 \
  --jq '[.. | objects | select(.type == "FRAME") | {id, name}]'
```

Comment creation and deletion are distinct write commands. They require `file_comments:write`, are never called from a read path, are not cached, and are not automatically retried.

## Output

- `json` is the default.
- `--compact` emits one-line JSON.
- `table` removes terminal control characters.
- `csv` prefixes spreadsheet-formula cells to prevent automatic execution.
- Table and CSV output require an array of objects.

## Cache and privacy

GET responses are cached for five minutes by default in `~/.cache/canvasprobe/`. Cache directories use mode `0700`; entries use `0600`. Cache identity includes the API origin, HTTP method, query, and a SHA-256 digest of the full token.

```sh
canvasprobe --no-cache file abc123
CANVASPROBE_CACHE_TTL=0 canvasprobe file abc123
canvasprobe cache purge
```

CanvasProbe has no service component and no telemetry. See [PRIVACY.md](PRIVACY.md) and [data handling](docs/data-handling.md).

## Security properties

- Production requests go only to `https://api.figma.com`.
- Cross-origin redirects and HTTPS downgrades are blocked before credentials can be forwarded.
- Write requests are explicit and are not automatically retried after ambiguous network failures.
- Configuration with group/other permissions is rejected.

Report vulnerabilities according to [SECURITY.md](SECURITY.md).

## API coverage

CanvasProbe deliberately covers a small set of endpoints and does not claim full API compatibility. See [API coverage](docs/api-coverage.md).

## License

Apache-2.0. See [LICENSE](LICENSE).
