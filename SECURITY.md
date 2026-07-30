# Security policy

## Supported versions

Until version 1.0, only the latest tagged release receives security fixes.

## Reporting

Use GitHub private vulnerability reporting. Do not open a public issue containing a token, private file key, raw design response, or exploit details.

Include:

- affected version and platform;
- minimal reproduction using synthetic data;
- expected impact;
- whether credentials or integration data may have been exposed.

## Response

The maintainer will acknowledge a report, preserve relevant evidence, assess whether credentials or integration data were affected, and coordinate any notifications required by applicable law or upstream developer terms. Public disclosure should wait until a fix or mitigation is available.

## Security model

- API credentials are sent only to the configured production origin.
- Production builds do not expose a base-URL override.
- Cross-origin redirects and HTTPS downgrades are blocked.
- Read responses may use a user-local permission-restricted cache.
- Write-back is limited to explicit comment commands and is never automatically retried.
