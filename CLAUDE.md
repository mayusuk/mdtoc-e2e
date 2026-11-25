# CLAUDE.md

- `specs/SPEC.md` is the source of truth. Don't change behaviour it doesn't describe.
- Go 1.25, standard library only, one module.
- Name tests after the requirement they prove: `TestFR2AC3_...`.
- Before finishing, run: `gofmt -l .` (must print nothing), `go vet ./...`, `go build ./...`, `go test -race -count=1 ./...`.
