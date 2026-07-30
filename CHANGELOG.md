# Changelog

All notable changes are documented here.

## Unreleased

## 0.2.0 - 2026-07-30

- Renamed the project, module, binary, configuration directory, cache directory, and environment prefix from CanvasProbe to FGM-C.

## 0.1.0 - 2026-07-30

- Renamed the original private project and binary to CanvasProbe.
- Replaced token command-line values with environment, protected config, or stdin input.
- Blocked credential forwarding across redirects.
- Isolated cache entries by full token digest and API origin.
- Removed corrupting client-side depth truncation.
- Limited `--fields` to flat component/style lists.
- Fixed compact JSON and hardened table/CSV output.
- Added explicit comment list/create/delete commands.
- Added cache lifecycle commands, privacy/security policies, CI, and release automation.
