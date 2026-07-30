# API coverage

CanvasProbe intentionally implements a narrow REST API surface:

| Command | Method and endpoint | Mutation |
|---|---|---|
| `me` | `GET /v1/me` | no |
| `file` | `GET /v1/files/:key` | no |
| `file ... nodes` | `GET /v1/files/:key/nodes` | no |
| `file ... components` | `GET /v1/files/:key/components` | no |
| `file ... styles` | `GET /v1/files/:key/styles` | no |
| `comments list` | `GET /v1/files/:key/comments` | no |
| `comments create` | `POST /v1/files/:key/comments` | yes |
| `comments delete` | `DELETE /v1/files/:key/comments/:comment_id` | yes |

The implementation does not claim coverage of projects, teams, variables, webhooks, images, exports, dev resources, analytics, or the full upstream OpenAPI schema.

API behavior and scopes should be reviewed against the official developer documentation before each release.
