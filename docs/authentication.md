# Authentication and scopes

CanvasProbe sends the user-provided credential in `X-Figma-Token` to the official REST API.

Recommended granular scopes:

| Command | Scope |
|---|---|
| `me` | `current_user:read` |
| `file`, `file ... nodes` | `file_content:read` |
| `file ... components/styles` | `library_content:read` |
| `comments list` | `file_comments:read` |
| `comments create/delete` | `file_comments:write` |

Scopes do not override file, project, team, or organization permissions.

Personal access tokens can be used for read and comment-write commands when granted the corresponding scopes. OAuth access tokens can also be supplied, but CanvasProbe does not implement an OAuth browser flow.

Plan access tokens can be used only for endpoints supported by that token category. In particular, Figma documents that plan access tokens do not support `file_comments:write` or `/v1/me`.

Primary input is `CANVASPROBE_TOKEN`; `FIGMA_API_TOKEN` is accepted as a descriptive upstream-compatible fallback. Avoid shell history and process arguments: use the environment, a `0600` config, or `--token-stdin`.
