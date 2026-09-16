# AGENTS.md — ai-tickets

MuxCore sidecar module (`ai-tickets`).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `ai-tickets` |
| Role | `ai` |
| Capability | `ai.tickets` |

## Build

```bash
cd ai-tickets
go test ./...
make lint
```

## Agent rules

- Modules run as gRPC sidecars; capabilities are the security boundary.
- TLS required in production (`MUXCORE_INSECURE_DISABLE_TLS` is dev-only).
- Match existing Go patterns; run `gofmt` and package tests before finishing.
- Roadmaps, task lists, and remaining-work checklists live in workspace [`MASTER-ROADMAP.md`](../MASTER-ROADMAP.md) and umbrella GitHub Issues. Do not add `ROADMAP.md` / `TASKS.md` in this repo.
