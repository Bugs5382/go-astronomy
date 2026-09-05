# go-astronomy

> Go astronomy library: observer-aware sun, moon, star, and constellation positions with twilight bands, arcs, and moon phases.

## Install

```bash
go get github.com/Bugs5382/go-astronomy
```

## Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

Commit discipline, AI-tell/emoji blocking, and the pre-push gofmt/vet/lint/test gate are enforced
by the governance hooks. Install them once per clone:

```bash
bash .claude/hooks/install.sh
```

## License

MIT (c) 2026 Shane
