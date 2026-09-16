# Governance

## Model

ibkrapi4go is a **single-maintainer** open-source project at present. The
maintainer has final say on design, scope, and releases. This section documents
how decisions are made so contributors know what to expect.

## Decision making

- **Small changes** (docs, tests, bug fixes): decided through pull-request review.
- **Design changes** (public API, architecture, dependencies): require an
  [ADR](./docs/adr/) merged before or alongside the code. ADRs are proposed via
  pull request and are accepted once the maintainer merges them.
- **Breaking changes**: follow [docs/RELEASING.md](./docs/RELEASING.md).

## Roles

| Role | Responsibility |
|------|----------------|
| Maintainer | final decisions, releases, security response, CODEOWNERS |
| Contributor | issues, pull requests, reviews (see CONTRIBUTING.md) |

Contributors who make sustained, high-quality contributions may be invited to
become reviewers or co-maintainers.

## Bus factor

The project currently has **one maintainer**. If you depend on it, consider
contributing. To reduce single-point risk, all decisions are recorded in the
repository (ADRs, issues, PRs) rather than in private channels.

## Conduct

All participation is governed by the [Code of Conduct](./CODE_OF_CONDUCT.md).

## Changes to this document

Changes to governance are made by pull request and require maintainer approval.
