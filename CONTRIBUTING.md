# Contributing

Before opening a pull request:

```sh
gofmt -w .
go mod tidy
go test ./...
go test -race ./...
go vet ./...
```

Keep changes small and preserve the security boundaries documented in `SECURITY.md`. Tests must use `httptest`; never use a real API token or live design file.

New write operations require an explicit subcommand, least-privilege scope documentation, no implicit retries, no response caching, and regression tests proving that read paths cannot trigger them.

By contributing, you agree that your contribution is licensed under Apache-2.0.
