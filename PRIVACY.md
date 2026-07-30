# Privacy

FGM-C is a local command-line tool. It has no project-operated server, telemetry, analytics, advertising, crash reporting, or third-party data transfer.

## Data processed

FGM-C processes the API token supplied by the user and responses returned by the Figma REST API. Responses may contain file metadata, design nodes, component and style names, user profile fields, and comments.

## Storage

GET responses may be stored in the user-local cache for the configured TTL. Write requests and responses are not cached. Tokens are not stored in cache entries or filenames; a SHA-256 digest participates in cache isolation.

The optional YAML configuration may store a token at the user's direction. FGM-C rejects configuration files accessible by group or other users.

## Control

Use `--no-cache` or `FGM_C_CACHE_TTL=0` to disable caching. Use `fgm-c cache purge` to delete all FGM-C cache entries.

Users are responsible for obtaining the rights and permissions required to access or modify data through their token. FGM-C processes data only to execute the command the user invokes.

## Contact

For a privacy or security concern, use GitHub's private vulnerability-reporting feature for this repository. Do not include API tokens or private design data in public issues.

Effective: 2026-07-30.
