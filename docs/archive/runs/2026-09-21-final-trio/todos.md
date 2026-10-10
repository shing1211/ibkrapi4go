# todos.md — Final Trio (Community + Examples + v1.0.1)

- **Run:** `2026-09-21-final-trio`
- **Mode:** BUILD

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|-----------|-----------|
| **v1.0.1 Release** |||||
| V1 | Update CHANGELOG: `[Unreleased]` → `[1.0.1]` with compare links | docs | todo | — | CHANGELOG has [1.0.1] section with compare links |
| V2 | Tag `v1.0.1` + push tag to both remotes | release | todo | V1 | Tag exists on both remotes |
| V3 | Patch GitHub release body from CHANGELOG | release | todo | V2 | Release body matches CHANGELOG |
| V4 | Create Gitee release entry (manual) | manual | todo | V2 | Gitee release created |
| **Community** |||||
| C1 | Add "good first issue" section to CONTRIBUTING.md | docs | todo | — | CONTRIBUTING.md updated |
| C2 | Post Discussions welcome announcement | docs | todo | C1 | Discussion post exists |
| C3 | Commit + push | release | todo | C1,C2 | Both remotes updated |
| **Examples** |||||
| E1 | Add `examples/mock/error-handling.go` | backend | todo | — | File exists, compiles, go vet passes |
| E2 | Add `examples/live/oauth2-flow.go` | backend | todo | — | File exists, compiles, go vet passes |
| E3 | Update `examples/README.md` with new entries | docs | todo | E1,E2 | examples/README.md updated |
| E4 | Commit + push | release | todo | E1,E2,E3 | Both remotes updated |
