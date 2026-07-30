## Change

Describe the smallest user-visible change.

## Verification

- [ ] `gofmt -w .`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] No token, private file key, or private design data is included
- [ ] New write behavior is explicit, uncached, and not automatically retried
