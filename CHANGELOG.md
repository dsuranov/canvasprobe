# Changelog

All notable changes are documented here.

## Unreleased

- Renamed the project and binary to CanvasProbe.
- Replaced token command-line values with environment, protected config, or stdin input.
- Blocked credential forwarding across redirects.
- Isolated cache entries by full token digest and API origin.
- Removed corrupting client-side depth truncation.
- Limited `--fields` to flat component/style lists.
- Fixed compact JSON and hardened table/CSV output.
- Added explicit comment list/create/delete commands.
- Added cache lifecycle commands, privacy/security policies, CI, and release automation.
