## Summary

<!-- What changed and why. -->

## Test plan

<!-- How you verified the change. For provider changes, note whether you ran
     the acceptance suite (`make testacc`) and against which image. -->

## Checklist

- [ ] `make build` and `make vet` pass
- [ ] `make lint` passes
- [ ] Unit tests added/updated where applicable (`make test`)
- [ ] Acceptance tests added/updated and run (`make testacc`) for new resources or data sources
- [ ] `make generate` run and committed, if the schema or examples changed (registry docs in `docs/`)
- [ ] Changes that consume new BridgePort API surface bumped the Go SDK dependency
